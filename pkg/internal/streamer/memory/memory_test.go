package memory

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func makeStepResults(n int) []*proto.StepResult {
	result := []*proto.StepResult{}
	for i := 0; i < n; i++ {
		result = append(result, &proto.StepResult{ExecResult: &proto.StepResult_ExecResult{ExitCode: int32(i)}})
	}
	return result
}

func Test_Streamer(t *testing.T) {
	var gotResults []*proto.StepResult

	numSourceResults := 5

	tests := map[string]struct {
		writer      func(*proto.StepResult) error
		validate    func(error)
		offset      int32
		wantResults int
	}{
		"happy path": {
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := New[*proto.StepResult]()
			gotResults = []*proto.StepResult{}
			wg := sync.WaitGroup{}

			wg.Add(1)
			var err error
			go func() {
				defer wg.Done()
				err = s.Follow(tt.offset, tt.writer)
			}()

			sourceResults := makeStepResults(numSourceResults)
			for _, result := range sourceResults {
				s.Write(result)
			}

			assert.Eventually(t, func() bool {
				return len(gotResults) == tt.wantResults
			}, 100*time.Millisecond, 25*time.Millisecond,
				"want: %d; got: %d", tt.wantResults, len(gotResults))

			s.Stop()
			wg.Wait()

			tt.validate(err)
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
		err := s.Follow(0, func(sr *proto.StepResult) error {
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
