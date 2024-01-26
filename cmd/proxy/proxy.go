package proxy

import (
	"fmt"
	"io"
	"net"
	"os"
)

// Note: proxying between stdin/out/err (the client) and gRPC (the server)

type Networking string

const (
	UnixSocket Networking = "unix"
	TCPAddress Networking = "tcp"
)

type Proxy struct {
	Address    string     `arg:"-a,--address" default:"127.0.0.1:8765" help:"host tcp networking address"`
	Socket     string     `arg:"-s,--socket" default:"/tmp/step-runner.sock" help:"unix domain socket path"`
	Networking Networking `arg:"-n,--networking" default:"unix" help:"networking type [unix,tcp]"`
}

func (p *Proxy) Run() error {
	protocol, address, err := p.netStuff()
	if err != nil {
		return err
	}

	return proxy(os.Stdout, os.Stdin, protocol, address)
}

func (p *Proxy) netStuff() (protocol, address string, err error) {
	switch p.Networking {
	case TCPAddress:
		return "tcp", p.Address, nil
	case UnixSocket:
		return "unix", p.Socket, nil
	}
	return "", "", fmt.Errorf("invalid networking %q", p.Networking)
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
