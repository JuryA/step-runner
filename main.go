package main

import (
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
)

type Runnable interface {
	Run() error
}

// sub-commands must satisfy the Runnable interface
type App struct {
	CI *ci.CI `arg:"subcommand:ci" help:"Run steps in a CI environment variable STEPS"`
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

	cmd, ok := p.Subcommand().(Runnable)
	if ok {
		return cmd.Run()
	}
	p.WriteHelp(os.Stderr)
	return nil
}
