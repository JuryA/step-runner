// package jobs implements
package jobs

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	rctx "gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/streamer/memory"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Job struct {
	TmpDir  string
	WorkDir string
	GlobCtx *rctx.Global    // To capture stdout/err from all subprocesses
	Ctx     context.Context // The context used to manage the Job's entire lifetime.
	ID      string          // The ID of the job to run/being run. Must be unique. Typically this will be the CI job ID.

	cancel     func()    // Used to cancel the Ctx.
	err        error     // Captures any error returned when executing steps.
	finished   bool      // Indicated whether all processing of this job has finished.
	finishTime time.Time // time when the job finished execution.
	mux        sync.RWMutex

	stepResults *memory.Streamer[*proto.StepResult]
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

	// TODO: add job timeout to RunRequest and hook it up here
	ctx, cancel := context.WithCancel(context.Background())

	globCtx, err := rctx.NewGlobal()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating global context: %w", err)
	}
	globCtx.WorkDir = workDir

	return &Job{
		TmpDir:      tmpDir,
		WorkDir:     workDir,
		ID:          request.Id,
		Ctx:         ctx,
		GlobCtx:     globCtx,
		cancel:      cancel,
		stepResults: memory.New[*proto.StepResult](),
	}, nil
}

func (j *Job) Err() error {
	j.mux.RLock()
	defer j.mux.RUnlock()
	return j.err
}

// Finish() finishes/completes natural job execution (i.e. not cancelled). It does not clean up resources created
// during job execution.
func (j *Job) Finish(err error) {
	now := time.Now()

	j.mux.Lock()
	defer j.mux.Unlock()
	if j.finished {
		return
	}

	j.err = err
	j.finished = true
	j.finishTime = now
	j.stepResults.Stop()
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
	j.Finish(j.Ctx.Err())
}

// StepResultWriter return a function that when called, will collect/accumulate StepResults into the embedded
// stepresult.Streamer instance.
func (j *Job) StepResultWriter() func(*proto.StepResult) { return j.stepResults.Write }

// FollowStepResults will write all current and future accumulated StepResults to the specified writer. This function
// returns:
// - a non-nil error returned by the writer.
// - the error that terminated the job (if any)
// - nil
func (j *Job) FollowStepResults(ctx context.Context, offset int32, writer func(*proto.StepResult) error) error {
	if err := j.stepResults.Follow(ctx, offset, writer); err != nil {
		return err
	}
	return j.Err()
}
