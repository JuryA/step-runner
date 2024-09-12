package bldr

import "gitlab.com/gitlab-org/step-runner/proto"

type ProtoSpecBuilder struct {
	outputSpec map[string]*proto.Spec_Content_Output
}

func ProtoSpec() *ProtoSpecBuilder {
	return &ProtoSpecBuilder{
		outputSpec: make(map[string]*proto.Spec_Content_Output),
	}
}

func (bldr *ProtoSpecBuilder) WithOutputSpec(outputSpec map[string]*proto.Spec_Content_Output) *ProtoSpecBuilder {
	bldr.outputSpec = outputSpec
	return bldr
}

func (bldr *ProtoSpecBuilder) Build() *proto.Spec {
	return &proto.Spec{
		Spec: &proto.Spec_Content{
			Inputs:       map[string]*proto.Spec_Content_Input{},
			Outputs:      bldr.outputSpec,
			OutputMethod: proto.OutputMethod_outputs,
		},
	}
}
