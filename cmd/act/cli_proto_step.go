package act

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type CLIStepsContext struct {
	StepsContext *proto.StepsContext
}

func (sc *CLIStepsContext) Set(input string) error {
	if err := protojson.Unmarshal([]byte(input), sc.StepsContext); err != nil {
		return fmt.Errorf("cannot unmarshal proto steps context: %w", err)
	}

	return nil
}

func (sc *CLIStepsContext) Type() string {
	return "proto steps context"
}

func (sc *CLIStepsContext) String() string {
	data, _ := protojson.Marshal(sc.StepsContext)
	return string(data)
}
