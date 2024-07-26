package domain

type Inputs struct {
	inputs []*Input
}

func NewInputs(inputs ...*Input) *Inputs {
	return &Inputs{
		inputs: inputs,
	}
}
