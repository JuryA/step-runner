package bldr

import "gitlab.com/gitlab-org/step-runner/proto"

type ProtoSpecDefinitionBuilder struct {
	spec       *proto.Spec
	definition *proto.Definition
}

func ProtoSpecDef() *ProtoSpecDefinitionBuilder {
	return &ProtoSpecDefinitionBuilder{
		spec:       ProtoSpec().Build(),
		definition: ProtoDef().Build(),
	}
}

func (bldr *ProtoSpecDefinitionBuilder) WithSpec(spec *proto.Spec) *ProtoSpecDefinitionBuilder {
	bldr.spec = spec
	return bldr
}

func (bldr *ProtoSpecDefinitionBuilder) WithDefinition(definition *proto.Definition) *ProtoSpecDefinitionBuilder {
	bldr.definition = definition
	return bldr
}

func (bldr *ProtoSpecDefinitionBuilder) Build() *proto.SpecDefinition {
	return &proto.SpecDefinition{
		Spec:       bldr.spec,
		Definition: bldr.definition,
		Dir:        "",
	}
}
