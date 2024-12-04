package act

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type CLIProtoStep struct {
	Step *proto.Step
}

func (s *CLIProtoStep) Set(input string) error {
	if err := protojson.Unmarshal([]byte(input), s.Step); err != nil {
		return fmt.Errorf("cannot unmarshal proto step: %w", err)
	}

	return nil
}

func (s *CLIProtoStep) Type() string {
	return "proto step"
}

func (s *CLIProtoStep) String() string {
	data, _ := protojson.Marshal(s.Step)
	return string(data)
}
