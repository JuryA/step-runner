package domain

type Outputs struct {
	outputs  []*Output
	delegate bool
}

func NewOutputs(delegate bool, outputs ...*Output) *Outputs {
	return &Outputs{
		delegate: delegate,
		outputs:  outputs,
	}
}
