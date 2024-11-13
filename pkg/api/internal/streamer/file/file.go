// package file implements the Streamer interface backed by a file buffer.
package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

type Streamer struct {
	io.WriteCloser
	cond     *sync.Cond
	filename string
	stop     bool
}

func New(filename string) (*Streamer, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("creating file %q: %w", filename, err)
	}
	return &Streamer{filename: filename, WriteCloser: f, cond: sync.NewCond(&sync.Mutex{})}, nil
}

func (s *Streamer) Write(p []byte) (int, error) {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	defer s.cond.Broadcast()
	return s.WriteCloser.Write(p)
}

func (s *Streamer) Stop() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	s.stop = true
	s.cond.Broadcast()
}

func (s *Streamer) Close() error {
	s.Stop()
	return s.WriteCloser.Close()
}

// toIOReader can be used to "cast" a func([]byte)(int, error) to an io.Reader.
type toIOReader func([]byte) (int, error)

func (r toIOReader) Read(p []byte) (int, error) { return r(p) }

func (s *Streamer) Follow(ctx context.Context, offset int64, writer io.Writer) error {
	f, err := os.OpenFile(s.filename, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("opening log file %q: %w", s.filename, err)
	}
	//nolint:errcheck
	defer f.Close()

	if _, err := f.Seek(offset, 0); err != nil {
		return fmt.Errorf("seeking log file %q: %w", s.filename, err)
	}

	scanner := bufio.NewScanner(toIOReader(func(p []byte) (int, error) {
		s.cond.L.Lock()
		defer s.cond.L.Unlock()
		for {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			n, err := f.Read(p)

			if err != io.EOF {
				return n, err
			}

			if s.stop {
				return n, io.EOF
			}

			s.cond.Wait()
		}
	}))

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data := scanner.Bytes()
		if len(data) == 0 {
			continue
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("streaming logs: %w", err)
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return fmt.Errorf("streaming logs: %w", err)
		}
	}
	// scanner.Err() returns nil instead of io.EOF even if an EOF stopped scanner.Scan().
	return scanner.Err()
}
