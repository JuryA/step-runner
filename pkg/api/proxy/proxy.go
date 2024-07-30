package proxy

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/sync/errgroup"
)

// Proxy connects the read and writer with the specified connection. Under typical usage the connection will be to the
// steps gRPC service listening on api.DefaultSocketPath, and the source and sink will be the stdin and stdout
// (respectively) of a streaming-text based protocol like `ssh` or `docker exec`.
//
// The proxy should exit immediately if either of the write or read loop finish. Whether proxying was complete and
// successful cannot be determined here and is up to the caller to determine based on the result of the data/operation
// being proxied.
func Proxy(source io.Reader, sink io.Writer, conn net.Conn) error {
	eg := errgroup.Group{}

	// pipe source to the connection
	eg.Go(func() error {
		defer conn.Close() // ensure the writing loop exits too...
		_, err := io.Copy(conn, source)
		if err != nil {
			return fmt.Errorf("proxying stdin to %q: %w", conn.RemoteAddr().String(), err)
		}
		return nil
	})

	// pipe the connection to sink
	eg.Go(func() error {
		defer conn.Close() // ensure the reading loop exists too...
		_, err := io.Copy(sink, conn)
		if err != nil {
			return fmt.Errorf("proxying %q to stdout: %w", conn.RemoteAddr().String(), err)
		}
		return nil
	})

	return eg.Wait()
}
