package service

import "gitlab.com/gitlab-org/step-runner/pkg/api/internal/jobs"

func (s *StepRunnerService) GetJob(id string) (*jobs.Job, bool) { return s.jobs.Get(id) }
