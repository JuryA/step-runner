package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/step-runner/proto"
	"golang.org/x/sync/errgroup"
)

func makeStepResults(n int) []*proto.StepResult {
	result := []*proto.StepResult{}
	for i := 0; i < n; i++ {
		result = append(result, &proto.StepResult{ExecResult: &proto.StepResult_ExecResult{ExitCode: int32(i)}})
	}
	return result
}

type testSpec struct {
	writer      func(*proto.StepResult) error
	validate    func(error)
	offset      int32
	wantResults int
	ctx         context.Context
	ctxCancel   func()
	wg          *sync.WaitGroup
}

func Test_Streamer(t *testing.T) {
	var gotResults []*proto.StepResult

	numSourceResults := 5

	tests := map[string]testSpec{
		"happy path": {
			ctx:         context.Background(),
			wantResults: numSourceResults,
			writer: func(sr *proto.StepResult) error {
				gotResults = append(gotResults, sr)
				return nil
			},
			validate: func(e error) {
				assert.NoError(t, e)
			},
		},
		"writer returns error": {
			ctx:         context.Background(),
			wantResults: 3,
			writer: func(sr *proto.StepResult) error {
				gotResults = append(gotResults, sr)
				if len(gotResults) < 3 {
					return nil
				}
				return errors.New("POW!!!")
			},
			validate: func(e error) {
				assert.ErrorContains(t, e, "POW!!!")
			},
		},
		"with offset": {
			ctx:         context.Background(),
			wantResults: 3,
			offset:      2,
			writer: func(sr *proto.StepResult) error {
				gotResults = append(gotResults, sr)
				return nil
			},
			validate: func(e error) {
				assert.NoError(t, e)
			},
		},
		"with offset greater than total results": {
			ctx:         context.Background(),
			wantResults: 0,
			offset:      6,
			writer: func(sr *proto.StepResult) error {
				gotResults = append(gotResults, sr)
				return nil
			},
			validate: func(e error) {
				assert.NoError(t, e)
			},
		},
		"context cancelled": func() testSpec {
			tt := testSpec{
				wantResults: 1,
				offset:      0,
				wg:          &sync.WaitGroup{},
				validate: func(e error) {
					assert.ErrorIs(t, e, context.Canceled)
				},
			}
			tt.wg.Add(1)
			tt.ctx, tt.ctxCancel = context.WithCancel(context.Background())
			tt.writer = func(sr *proto.StepResult) error {
				defer tt.wg.Done()
				defer tt.ctxCancel()
				gotResults = append(gotResults, sr)
				return nil
			}

			return tt
		}(),
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := New[*proto.StepResult]()
			gotResults = []*proto.StepResult{}

			errs := errgroup.Group{}
			errs.Go(func() error {
				return s.Follow(tt.ctx, tt.offset, tt.writer)
			})

			sourceResults := makeStepResults(numSourceResults)
			for _, result := range sourceResults {
				s.Write(result)
				if tt.wg != nil {
					tt.wg.Wait()
				}
			}

			assert.Eventually(t, func() bool {
				return len(gotResults) == tt.wantResults
			}, 100*time.Millisecond, 25*time.Millisecond)

			s.Stop()

			tt.validate(errs.Wait())
			for i := range gotResults {
				assert.Equal(t, sourceResults[i+int(tt.offset)].ExecResult.ExitCode, gotResults[i].ExecResult.ExitCode)
			}
		})
	}
}

func Test_Streamer_StopBeforeFollow(t *testing.T) {
	s := New[*proto.StepResult]()
	gotResults := []*proto.StepResult{}
	wg := sync.WaitGroup{}

	sourceResults := makeStepResults(5)
	for _, result := range sourceResults {
		s.Write(result)
	}
	s.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Follow(context.Background(), 0, func(sr *proto.StepResult) error {
			gotResults = append(gotResults, sr)
			return nil
		})
		assert.NoError(t, err)
	}()

	assert.Eventually(t, func() bool {
		return len(gotResults) == len(sourceResults)
	}, 100*time.Millisecond, 25*time.Millisecond)

	wg.Wait()

	for i := range gotResults {
		assert.Equal(t, sourceResults[i].ExecResult.ExitCode, gotResults[i].ExecResult.ExitCode)
	}
}
