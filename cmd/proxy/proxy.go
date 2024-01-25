package proxy

import (
	"fmt"
	"io"
	"net"
	"os"
)

// Note: proxying between stdin/out/err (the client) and gRPC (the server)

type Proxy struct{}

func (p *Proxy) Run() error {
	return proxy(os.Stdout, os.Stdin, "tcp", "localhost:8765")
}

// proxy connects the read and writer with the dialed connection.
func proxy(w io.Writer, r io.Reader, network, address string) error {
	conn, err := net.Dial(network, address)
	if err != nil {
		return fmt.Errorf("proxy dialing: %w", err)
	}

	errChan := make(chan error)
	// pipe stdin to the connection
	go func() {
		_, err := io.Copy(conn, r)
		if err != nil {
			err = fmt.Errorf("proxy r to conn: %w", err)
		}
		errChan <- err
	}()

	// pipe the connection to stdout
	go func() {
		_, err := io.Copy(w, conn)
		if err != nil {
			err = fmt.Errorf("proxy w to conn: %w", err)
		}
		errChan <- err
	}()

	err1 := <-errChan
	err2 := <-errChan
	err3 := conn.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}
