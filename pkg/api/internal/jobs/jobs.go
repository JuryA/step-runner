// package jobs implements
package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
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
	Ctx     context.Context // The context used to manage the Job's entire lifetime.
	ID      string          // The ID of the job to run/being run. Must be unique. Typically this will be the CI job ID.

	cancel     func()    // Used to cancel the Ctx.
	err        error     // Captures any error returned when executing steps.
	startTime  time.Time // time when the job finished execution.
	finishTime time.Time // time when the job finished execution.
	mux        sync.RWMutex
	finishC    chan struct{}
	runOnce    sync.Once
	closeOnce  sync.Once

	logs *file.Streamer

	status proto.StepResult_Status
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

	return &Job{
		TmpDir:  tmpDir,
		WorkDir: workDir,
		ID:      request.Id,
		Ctx:     ctx,
		cancel:  cancel,
		logs:    logs,
		status:  proto.StepResult_unspecified,
		finishC: make(chan struct{}, 1),
	}, nil
}

// Run actually starts execution of the steps request and captures the result. It is intended to be run in a
// goroutine.
func (j *Job) Run(stepsCtx *runner.StepsContext, step runner.Step) {
	j.runOnce.Do(func() {
		defer func() {
			stepsCtx.Cleanup()
			j.logs.Stop()
			_ = j.logs.Close()
			j.finishC <- struct{}{}
		}()

		stepsCtx.WorkDir = j.WorkDir
		// TODO: differentiate between stdin/stderr
		stepsCtx.Stderr = j.logs
		stepsCtx.Stdout = j.logs

		j.mux.Lock()

		if j.Ctx.Err() != nil {
			j.err = fmt.Errorf("job %q cancelled before execution started", j.ID)
			j.status = proto.StepResult_cancelled
			j.mux.Unlock()
			log.Println(j.err.Error())
			return
		}

		j.startTime = time.Now()
		j.status = proto.StepResult_running
		j.mux.Unlock()

		result, err := step.Run(j.Ctx, stepsCtx)

		j.mux.Lock()
		defer j.mux.Unlock()

		j.err = err
		j.finishTime = time.Now()
		j.status = j.computeFinalStatus(result, err)

		if err != nil {
			// TODO: better logging
			log.Printf("an error occurred executing job %q: %s", j.ID, err)
		}
	})
}

func (j *Job) computeFinalStatus(stepResult *proto.StepResult, err error) proto.StepResult_Status {
	// take the status of the root step-result as the overall execution status.
	switch stepResult.Status {
	case proto.StepResult_unspecified, proto.StepResult_running:
		log.Printf("invalid final status %q for job %q", stepResult.Status.String(), j.ID)
		return proto.StepResult_failure
	case proto.StepResult_failure:
		// When a job is cancelled (by calling `Job.Close()`) or times out (both of
		// which cancel the context passed to `exec.CommandContext()`), the returned
		// error can:
		//  * be one of context.Cancelled or context.DeadlineExceeded.
		//  * be another error type that ends with the string "signal: killed".
		//
		// In both cases the `StepResult_Status` returned by `Step.Run()` is
		// `failure`, but we want it to be `cancelled`. Since the latter can also
		// happen when the process is otherwise killed (e.g. OOM killer), so we
		// have to also check that the context was actually cancelled.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			(j.Ctx.Err() != nil && strings.HasSuffix(err.Error(), "signal: killed")) {
			return proto.StepResult_cancelled
		}
		fallthrough
	default:
		return stepResult.Status
	}
}

// Close() cancels jobs (if still running) and cleans up all resources associated with managing the job.
func (j *Job) Close() {
	j.closeOnce.Do(func() {
		j.cancel()
		// block until Run has exited

		select {
		case <-j.finishC:
		case <-time.NewTimer(time.Second * 2).C:
			// A caller called close without first calling Run... 2 seconds ought to be enough for exec.Cmd.Run() to
			// return...
			j.mux.Lock()
			defer j.mux.Unlock()
			if j.status == proto.StepResult_unspecified {
				j.status = proto.StepResult_cancelled
			}
			if j.err == nil {
				j.err = context.Canceled
			}
		}

		_ = os.RemoveAll(j.TmpDir)
	})
}

// FollowLogs writes the stdout/stderr captured from running steps to the supplied writer.
// The function blocks until the steps are run and logs are written to the writer.
// Errors returned are indicative of failure to write to the writer, not failure of running steps.
func (j *Job) FollowLogs(ctx context.Context, offset int64, writer io.Writer) error {
	if err := j.logs.Follow(ctx, offset, writer); err != nil {
		return fmt.Errorf("following logs for job %q: %w", j.ID, err)
	}
	return nil
}

// Status returns a proto.Status objected representing the current status of the job. If the job has not Finished, some
// fields may be empty/nil.
func (j *Job) Status() *proto.Status {
	j.mux.RLock()
	defer j.mux.RUnlock()

	st := proto.Status{
		Id:     j.ID,
		Status: j.status,
	}

	if !j.startTime.IsZero() {
		st.StartTime = timestamppb.New(j.startTime)
	}

	if !j.finishTime.IsZero() {
		st.EndTime = timestamppb.New(j.finishTime)
	}
	if j.err != nil {
		st.Message = j.err.Error()
	}
	return &st
}
