// package jobs implements
package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/streamer/file"
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
	startTime  time.Time // time when the job finished execution.
	finishTime time.Time // time when the job finished execution.
	mux        sync.RWMutex

	// TODO: This is temporary, until we implement streaming of step-results.
	stepResult *proto.StepResult
	logs       *file.Streamer

	stepResultStatus proto.StepResult_Status
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
		_ = os.RemoveAll(tmpDir)
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
		TmpDir:           tmpDir,
		WorkDir:          workDir,
		ID:               request.Id,
		Ctx:              ctx,
		GlobCtx:          globCtx,
		startTime:        time.Now(),
		cancel:           cancel,
		logs:             logs,
		stepResultStatus: proto.StepResult_running,
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

	j.logs.Stop()
	_ = j.logs.Close()

	j.stepResultStatus = computeFinalStatus(result, err)
}

// TODO: this temporary until we add step-result streaming
func computeFinalStatus(stepResult *proto.StepResult, err error) proto.StepResult_Status {
	if stepResult == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return proto.StepResult_cancelled
	}

	// take the state of the root step-result as the overall execution state.
	return stepResult.Status
}

func (j *Job) Finished() bool {
	j.mux.RLock()
	defer j.mux.RUnlock()
	return j.finished
}

// Close() cancels jobs (if still running) and cleans up all resources associated with managing the job.
func (j *Job) Close() {
	j.cancel()
	//nolint:errcheck
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

// Status returns a proto.Status objected representing the current status of the job. If the job has not Finished, some
// fields may be empty/nil.
func (j *Job) Status() *proto.Status {
	j.mux.RLock()
	defer j.mux.RUnlock()

	st := proto.Status{
		Id:        j.ID,
		StartTime: timestamppb.New(j.startTime),
		Status:    j.stepResultStatus,
	}

	if j.finished {
		st.EndTime = timestamppb.New(j.finishTime)
	}
	if j.err != nil {
		st.Message = j.err.Error()
	}
	return &st
}
