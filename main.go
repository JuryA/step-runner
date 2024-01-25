package main

import (
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/proxy"
	"gitlab.com/gitlab-org/step-runner/cmd/service"
)

type App struct {
	CI    *ci.CI         `arg:"subcommand:ci" help:"Run steps in a CI environment variable STEPS"`
	Serve *service.Serve `arg:"subcommand:serve" help:"Run step-runner server"`
	Proxy *proxy.Proxy   `arg:"subcommand:proxy" help:"proxy gRPC calls from stdin to a local steps-runner service and back to stdout"`
}

func (App) Description() string {
	return "Step Runner executes a series of CI steps"
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	app := App{}
	p := arg.MustParse(&app)

	switch {
	case app.CI != nil:
		return app.CI.Run()
	case app.Serve != nil:
		return app.Serve.Run()
	case app.Proxy != nil:
		return app.Proxy.Run()
	default:
		p.WriteHelp(os.Stderr)
	}
	return nil
}
