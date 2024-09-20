package bldr

import "gitlab.com/gitlab-org/step-runner/proto"

type ProtoDefinitionBuilder struct {
	defType proto.DefinitionType
	env     map[string]string
	exec    *proto.Definition_Exec
}

func ProtoDef() *ProtoDefinitionBuilder {
	return &ProtoDefinitionBuilder{
		defType: proto.DefinitionType_exec,
		env:     map[string]string{},
		exec: &proto.Definition_Exec{
			Command: []string{"bash", "-c", "echo 'hello world'"},
			WorkDir: "",
		},
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

func (bldr *ProtoDefinitionBuilder) Build() *proto.Definition {
	return &proto.Definition{
		Type:     bldr.defType,
		Exec:     bldr.exec,
		Steps:    nil,
		Outputs:  nil,
		Env:      bldr.env,
		Delegate: "",
	}
}
