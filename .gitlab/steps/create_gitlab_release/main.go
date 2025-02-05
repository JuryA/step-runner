package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"time"

	"gitlab.com/gitlab-org/step-runner/steps/create_gitlab_release/pkg/gitlab"
)

var projectID = flag.String("projectID", "", "")
var jobToken = flag.String("jobToken", "", "")
var commit = flag.String("commit", "", "")
var description = flag.String("description", "", "")
var tag = flag.String("tag", "", "")
var timeoutSecs = flag.Int("timeoutInSeconds", 0, "")

func main() {
	if err := validateConfig(); err != nil {
		log.Fatalln(err)
	}

	if err := release(); err != nil {
		log.Fatalln(err)
	}
}

func release() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(*timeoutSecs))
	defer cancel()

	repo := gitlab.NewProject(*projectID, *jobToken)
	return repo.Release(ctx, *commit, *tag, *description)
}

func validateConfig() error {
	flag.Parse()

	switch {
	case projectID == nil || *projectID == "":
		return errors.New("projectID is required")
	case jobToken == nil || *jobToken == "":
		return errors.New("jobToken is required")
	case commit == nil || *commit == "":
		return errors.New("commit is required")
	case tag == nil || *tag == "":
		return errors.New("tag is required")
	case timeoutSecs == nil || *timeoutSecs <= 0:
		return errors.New("timeoutInSeconds must be a positive integer")
	case description == nil:
		return errors.New("description is required")
	}

	return nil
}
