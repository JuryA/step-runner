package domain

type Input struct {
	name         string
	sensitive    bool
	valueType    any
	valueDefault any
}

func NewInput(name string, valueType, valueDefault any, sensitive bool) *Input {
	return &Input{
		name:         name,
		valueType:    valueType,
		valueDefault: valueDefault,
		sensitive:    sensitive,
	}
}
