package file

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
)

var data = [][]byte{
	[]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"),
	[]byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"),
	[]byte("cccccccccccccccccccccccccccccccccc\n"),
}

type testSpec struct {
	writer      func(*testSpec, []byte) (int, error)
	validate    func(*testSpec, error)
	offset      int64
	buf         test.SyncBuff
	wantWritten []byte
	ctx         context.Context
	ctxCancel   func()
}

func (tt *testSpec) Write(p []byte) (int, error) {
	return tt.writer(tt, p)
}

func Test_Streamer(t *testing.T) {
	tests := map[string]*testSpec{
		"happy path": {
			ctx:         context.Background(),
			wantWritten: bytes.Join(data, nil),
			writer: func(tt *testSpec, p []byte) (int, error) {
				return tt.buf.Write(p)
			},
			validate: func(tt *testSpec, e error) {
				assert.NoError(t, e)
				// all the data was streamed though...
				assert.Equal(t, string(tt.wantWritten), tt.buf.String())
			},
		},
		"writer returns error": {
			ctx:         context.Background(),
			wantWritten: data[0][:len(data[0])-1],
			writer: func(tt *testSpec, p []byte) (int, error) {
				_, _ = tt.buf.Write(p)
				return 0, errors.New("POW!!!")
			},
			validate: func(tt *testSpec, e error) {
				assert.ErrorContains(t, e, "POW!!!")
				// only the first write (less the newline) was streamed though...
				assert.Equal(t, string(tt.wantWritten), tt.buf.String())
			},
		},
		"with offset": {
			ctx:         context.Background(),
			offset:      11,
			wantWritten: bytes.Join(data, nil)[11:],
			writer: func(tt *testSpec, p []byte) (int, error) {
				return tt.buf.Write(p)
			},
			validate: func(tt *testSpec, e error) {
				assert.NoError(t, e)
				// all the data after the offset was streamed though...
				assert.Equal(t, string(tt.wantWritten), tt.buf.String())
			},
		},
		"with offset greater than total written": {
			ctx:         context.Background(),
			offset:      150,
			wantWritten: []byte{},
			writer: func(tt *testSpec, p []byte) (int, error) {
				return tt.buf.Write(p)
			},
			validate: func(tt *testSpec, e error) {
				assert.NoError(t, e)
				// all the data after the offset was streamed though...
				assert.Equal(t, string(tt.wantWritten), tt.buf.String())
			},
		},
		"context cancelled ": func() *testSpec {
			tt := testSpec{
				wantWritten: data[0],
				writer: func(tt *testSpec, p []byte) (int, error) {
					defer tt.ctxCancel()
					return tt.buf.Write(p)
				},
				validate: func(tt *testSpec, e error) {
					assert.ErrorIs(t, e, context.Canceled)
					// only the first write was streamed though...
					assert.Equal(t, string(tt.wantWritten), tt.buf.String())
				},
			}

			tt.ctx, tt.ctxCancel = context.WithCancel(context.Background())
			return &tt
		}(),
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filename := test.TestDirName(t)
			s, err := New(filename)
			require.NoError(t, err)
			defer os.Remove(filename)
			defer s.Close()

			errs := errgroup.Group{}
			errs.Go(func() error {
				return s.Follow(tt.ctx, tt.offset, tt)
			})

			for _, p := range data {
				_, err := s.Write(p)
				assert.NoError(t, err)
			}

			written := 0
			assert.Eventually(t, func() bool {
				defer func() { written = tt.buf.Len() }()
				return written == tt.buf.Len()
			}, 100*time.Millisecond, 25*time.Millisecond)

			s.Stop()
			tt.validate(tt, errs.Wait())
		})
	}
}

func Test_Streamer_StopBeforeFollow(t *testing.T) {
	filename := test.TestDirName(t)
	s, err := New(filename)
	require.NoError(t, err)
	defer os.Remove(filename)
	defer s.Close()

	buf := test.SyncBuff{}

	for _, p := range data {
		_, err := s.Write(p)
		assert.NoError(t, err)
	}
	s.Stop()

	errs := errgroup.Group{}
	errs.Go(func() error { return s.Follow(context.Background(), 0, &buf) })

	written := 0
	assert.Eventually(t, func() bool {
		defer func() { written = buf.Len() }()
		return written == buf.Len()
	}, 100*time.Millisecond, 25*time.Millisecond)

	assert.NoError(t, errs.Wait())
	assert.Equal(t, string(bytes.Join(data, nil)), buf.String())
}

func Test_Streamer_MultipleFollowers(t *testing.T) {
	filename := test.TestDirName(t)
	s, err := New(filename)
	require.NoError(t, err)
	defer os.Remove(filename)
	defer s.Close()

	numFollowers := 5
	bufs := []*test.SyncBuff{}
	errs := errgroup.Group{}

	for i := 0; i < numFollowers; i++ {
		b := test.SyncBuff{}
		bufs = append(bufs, &b)
		errs.Go(func() error {
			return s.Follow(context.Background(), 0, &b)
		})
	}

	for _, p := range data {
		_, err := s.Write(p)
		assert.NoError(t, err)
	}

	// wait for all followers to start
	assert.Eventually(t, func() bool { return len(bufs) == numFollowers }, 200*time.Millisecond, 25*time.Millisecond)
	b0 := bufs[0]

	written := 0
	// wait for all followers to read all data
	assert.Eventually(t, func() bool {
		defer func() { written = b0.Len() }()
		return written == b0.Len()
	}, 100*time.Millisecond, 25*time.Millisecond)

	s.Stop()

	assert.NoError(t, errs.Wait())
	assert.Equal(t, string(bytes.Join(data, nil)), b0.String())

	for _, bi := range bufs {
		assert.Equal(t, b0.String(), bi.String())
	}
}
