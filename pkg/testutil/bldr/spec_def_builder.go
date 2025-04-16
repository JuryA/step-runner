package bldr

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type SpecDefinitionBuilder struct {
	spec       *proto.Spec
	definition *proto.Definition
}

func SpecDef() *SpecDefinitionBuilder {
	return &SpecDefinitionBuilder{
		spec:       ProtoSpec().Build(),
		definition: ProtoDef().Build(),
	}
}

func (bldr *SpecDefinitionBuilder) WithSpec(spec *proto.Spec) *SpecDefinitionBuilder {
	bldr.spec = spec
	return bldr
}

func (bldr *SpecDefinitionBuilder) WithDefinition(definition *proto.Definition) *SpecDefinitionBuilder {
	bldr.definition = definition
	return bldr
}

func (bldr *SpecDefinitionBuilder) Build() *runner.SpecDefinition {
	return runner.NewSpecDefinition(bldr.spec, bldr.definition, "")
}
