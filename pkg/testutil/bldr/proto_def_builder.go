package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoDefinitionBuilder struct {
	defType proto.DefinitionType
	env     map[string]string
	exec    *proto.Definition_Exec
	outputs map[string]*structpb.Value
}

func ProtoDef() *ProtoDefinitionBuilder {
	return &ProtoDefinitionBuilder{
		defType: proto.DefinitionType_exec,
		env:     map[string]string{},
		exec: &proto.Definition_Exec{
			Command: []string{"bash", "-c", "echo 'hello world'"},
			WorkDir: "",
		},
		outputs: map[string]*structpb.Value{},
	}
}

func (bldr *ProtoDefinitionBuilder) WithEnvVar(name, value string) *ProtoDefinitionBuilder {
	bldr.env[name] = value
	return bldr
}

func (bldr *ProtoDefinitionBuilder) WithExecType(workDir string, command []string) *ProtoDefinitionBuilder {
	bldr.defType = proto.DefinitionType_exec
	bldr.exec = &proto.Definition_Exec{Command: command, WorkDir: workDir}
	return bldr
}

func (bldr *ProtoDefinitionBuilder) WithOutput(name string, value *structpb.Value) *ProtoDefinitionBuilder {
	bldr.outputs[name] = value
	return bldr
}

func (bldr *ProtoDefinitionBuilder) Build() *proto.Definition {
	return &proto.Definition{
		Type:     bldr.defType,
		Exec:     bldr.exec,
		Steps:    nil,
		Outputs:  bldr.outputs,
		Env:      bldr.env,
		Delegate: "",
	}
}
