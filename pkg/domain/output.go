package domain

type Output struct {
	name         string
	sensitive    bool
	valueDefault any
	valueType    any
}

func NewOutput(name string, valueType, valueDefault any, sensitive bool) *Output {
	return &Output{
		name:         name,
		valueType:    valueType,
		valueDefault: valueDefault,
		sensitive:    sensitive,
	}
}
