package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nektos/act/pkg/model"
	act_runner "github.com/nektos/act/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

func Run(action, image string, inputs map[string]string) (*proto.StepResult, error) {
	var (
		dir string
		err error
	)
	if isLocal(action) {
		dir = action
	} else {
		dir, err = os.MkdirTemp("", "")
		if err != nil {
			return nil, fmt.Errorf("making temp dir for workflow: %w", err)
		}
		defer os.RemoveAll(dir)
	}
	expectedOutputs, err := getActionOutputs(dir, action)
	if err != nil {
		return nil, fmt.Errorf("getting action outputs: %w", err)
	}
	workflowFilename, err := createWorkflow(dir, action, inputs, expectedOutputs)
	if err != nil {
		return nil, fmt.Errorf("creating single action workflow: %w", err)
	}
	outputs, err := runWorkflowJob(
		context.Background(), // TODO handle signals in main.go to shutdown gracefully
		image,
		workflowFilename,
		"single-action",
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("running action: %q %w", action, err)
	}
	stepResult := &proto.StepResult{
		Outputs: map[string]*structpb.Value{},
		Exports: map[string]string{},
	}
	for k, v := range outputs {
		stepResult.Outputs[k] = structpb.NewStringValue(v)
	}
	return stepResult, nil
}

func getActionOutputs(dir, action string) (map[string]model.Output, error) {
	if !isLocal(action) {
		repo, ref, ok := strings.Cut(action, "@")
		if !ok {
			return nil, fmt.Errorf("want url@ref. got %q", action)
		}
		url := "https://github.com/" + repo
		_, err := git.PlainClone(dir, false, &git.CloneOptions{
			Depth:             1,
			SingleBranch:      true,
			RecurseSubmodules: git.SubmoduleRescursivity(1),
			URL:               url,
			ReferenceName:     plumbing.ReferenceName(ref),
		})
		if err != nil {
			return nil, fmt.Errorf("cloning action %q: %w", url, err)
		}
	}
	actionFile := filepath.Join(dir, "action.yml")
	bytes, err := os.ReadFile(actionFile)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", actionFile, err)
	}
	actionModel := model.Action{}
	err = yaml.Unmarshal(bytes, &actionModel)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling action.yml: %w", err)
	}
	return actionModel.Outputs, nil
}

func createWorkflow(
	dir string,
	action string,
	inputs map[string]string,
	expectedOutputs map[string]model.Output,
) (string, error) {
	w := newWorkflow(action, inputs, expectedOutputs)
	data, err := yaml.Marshal(w)
	if err != nil {
		return "", fmt.Errorf("marshaling workflow: %w", err)
	}
	workflowFilename := filepath.Join(dir, "workflow.yml")
	err = os.WriteFile(workflowFilename, data, 0600)
	if err != nil {
		return "", fmt.Errorf("writing workflow file: %w", err)
	}
	return workflowFilename, nil
}

func runWorkflowJob(
	ctx context.Context,
	image string,
	workflowFilename,
	jobID string,
	inputs map[string]string,
) (map[string]string, error) {
	planner, err := model.NewWorkflowPlanner(workflowFilename, true)
	if err != nil {
		return nil, fmt.Errorf("creating workflow planner: %w", err)
	}
	plan, err := planner.PlanJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("planning job %q: %w", jobID, err)
	}
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	env := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			env[k] = v
		}
	}
	docker := "unix:///var/run/docker.sock"
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		docker = dockerHost
	}
	config := &act_runner.Config{
		Actor:                 "gitlab.com/components/action-runner",
		EventName:             "push",
		ForcePull:             true,
		ForceRebuild:          true,
		Workdir:               workDir,
		ActionCacheDir:        filepath.Join(workDir, ".cache", "act"),
		ActionOfflineMode:     false,
		BindWorkdir:           true,
		LogOutput:             true,
		Env:                   env,
		Inputs:                inputs,
		Platforms:             map[string]string{"act-image": image},
		Privileged:            false,
		ContainerDaemonSocket: docker,
		UseGitIgnore:          true,
		GitHubInstance:        "github.com",
		RemoteName:            "origin",
		ContainerNetworkMode:  "host",
	}
	r, err := act_runner.New(config)
	if err != nil {
		return nil, fmt.Errorf("creating runner: %w", err)
	}
	executor := r.NewPlanExecutor(plan)
	err = executor(ctx)
	if err != nil {
		return nil, fmt.Errorf("executing: %w", err)
	}
	if len(plan.Stages) != 1 {
		return nil, fmt.Errorf("reading outputs: expecting exactly 1 stage")
	}
	if len(plan.Stages[0].Runs) != 1 {
		return nil, fmt.Errorf("reading outputs: expecting exactly 1 run")
	}
	if plan.Stages[0].Runs[0].Workflow == nil {
		return nil, fmt.Errorf("reading outputs: nil workflow")
	}
	job, ok := plan.Stages[0].Runs[0].Workflow.Jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("reading outputs: could not find job %q", jobID)
	}
	return job.Outputs, nil
}

func isLocal(action string) bool {
	return strings.HasPrefix(action, ".") || strings.HasPrefix(action, "/")
}
