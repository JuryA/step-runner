package bldr

import "gitlab.com/gitlab-org/step-runner/proto"

type ProtoSpecBuilder struct {
	outputMethod proto.OutputMethod
	outputSpec   map[string]*proto.Spec_Content_Output
}

func ProtoSpec() *ProtoSpecBuilder {
	return &ProtoSpecBuilder{
		outputMethod: proto.OutputMethod_outputs,
		outputSpec:   make(map[string]*proto.Spec_Content_Output),
	}
}

func (bldr *ProtoSpecBuilder) WithOutputSpec(outputSpec map[string]*proto.Spec_Content_Output) *ProtoSpecBuilder {
	bldr.outputSpec = outputSpec
	return bldr
}

func (bldr *ProtoSpecBuilder) WithOutputMethod(outputMethod proto.OutputMethod) *ProtoSpecBuilder {
	bldr.outputMethod = outputMethod
	return bldr
}

func (bldr *ProtoSpecBuilder) Build() *proto.Spec {
	return &proto.Spec{
		Spec: &proto.Spec_Content{
			Inputs:       map[string]*proto.Spec_Content_Input{},
			Outputs:      bldr.outputSpec,
			OutputMethod: bldr.outputMethod,
		},
	}
}
