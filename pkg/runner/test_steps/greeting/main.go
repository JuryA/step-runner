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
	os.WriteFile(os.Getenv("STEP_RUNNER_OUTPUT"), []byte("name="+*name), 0640)
	os.WriteFile(os.Getenv("STEP_RUNNER_ENV"), []byte("NAME="+*name), 0640)
}
