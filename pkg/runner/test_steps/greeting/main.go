package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	name     = flag.String("name", "", "")
	greeting = flag.String("greeting", "", "")
)

func main() {
	flag.Parse()
	fmt.Println(*greeting)
	if err := os.WriteFile(os.Getenv("STEP_RUNNER_OUTPUT"), []byte(fmt.Sprintf("name=%q", *name)), 0640); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Getenv("STEP_RUNNER_ENV"), []byte(fmt.Sprintf("NAME=%q", *name)), 0640); err != nil {
		panic(err)
	}
}
