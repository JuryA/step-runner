// package jobs implements
package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/streamer/file"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Job struct {
	TmpDir  string
	WorkDir string
	GlobCtx *runner.GlobalContext // To capture stdout/err from all subprocesses
	Ctx     context.Context       // The context used to manage the Job's entire lifetime.
	ID      string                // The ID of the job to run/being run. Must be unique. Typically this will be the CI job ID.

	cancel     func()    // Used to cancel the Ctx.
	err        error     // Captures any error returned when executing steps.
	finished   bool      // Indicated whether all processing of this job has finished.
	finishTime time.Time // time when the job finished execution.
	mux        sync.RWMutex

	// TODO: This is temporary, until we implement streaming of step-results.
	stepResult *proto.StepResult
	logs       *file.Streamer
}

func New(request *proto.RunRequest) (*Job, error) {
	workDir := request.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	if request.Job != nil && request.Job.BuildDir != "" {
		workDir = request.Job.BuildDir
	}

	if err := os.MkdirAll(workDir, 0700); err != nil {
		return nil, fmt.Errorf("creating workdir %q: %w", workDir, err)
	}

	tmpDir := path.Join(os.TempDir(), "step-runner-output-"+request.Id)
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return nil, fmt.Errorf("creating tmpdir %q: %w", tmpDir, err)
	}

	logs, err := file.New(path.Join(tmpDir, "logs"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("creating log file: %w", err)
	}

	// TODO: add job timeout to RunRequest and hook it up here
	ctx, cancel := context.WithCancel(context.Background())

	globCtx, err := runner.NewGlobalContext()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating global context: %w", err)
	}
	globCtx.WorkDir = workDir
	// TODO: differentiate between stdin/stderr
	globCtx.Stderr = logs
	globCtx.Stdout = logs

	return &Job{
		TmpDir:  tmpDir,
		WorkDir: workDir,
		ID:      request.Id,
		Ctx:     ctx,
		GlobCtx: globCtx,
		cancel:  cancel,
		logs:    logs,
	}, nil
}

// Result returns the StepResult and error resulting from executing the step. Note that this function might not be
// necessary when we implement streaming step-results as they are executed.
func (j *Job) Result() (*proto.StepResult, error) {
	j.mux.RLock()
	defer j.mux.RUnlock()
	return j.stepResult, j.err
}

// Finish() finishes/completes natural job execution (i.e. not cancelled). It does not clean up resources created
// during job execution. The signature of this method may change when we implement streaming step-results.
func (j *Job) Finish(result *proto.StepResult, err error) {
	now := time.Now()

	j.mux.Lock()
	defer j.mux.Unlock()
	if j.finished {
		return
	}

	j.stepResult = result
	j.err = err
	j.finished = true
	j.finishTime = now

	if err != nil {
		j.stepResult = nil
	}
	j.logs.Stop()
	j.logs.Close()
}

func (j *Job) Finished() bool {
	j.mux.RLock()
	defer j.mux.RUnlock()
	return j.finished
}

// Close() cancels jobs (if still running) and cleans up all resources associated with managing the job.
func (j *Job) Close() {
	j.cancel()
	defer os.RemoveAll(j.TmpDir)
	defer j.GlobCtx.Cleanup()
	j.Finish(nil, j.Ctx.Err())
}

// FollowLogs will write all current and future accumulated logs written to stdout/stderr from all step sub-processes to
// the specified writer. This function returns:
// - a non-nil error returned by the writer.
// - the error that terminated the job (if any)
// - nil
func (j *Job) FollowLogs(ctx context.Context, offset int64, writer io.Writer) error {
	if err := j.logs.Follow(ctx, offset, writer); err != nil {
		return err
	}
	return j.err
}
