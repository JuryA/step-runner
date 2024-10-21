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
	if err := os.WriteFile(os.Getenv("STEP_RUNNER_OUTPUT"), []byte(fmt.Sprintf(`name="%s"`, *name)), 0640); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Getenv("STEP_RUNNER_ENV"), []byte("NAME="+*name), 0640); err != nil {
		panic(err)
	}
}
