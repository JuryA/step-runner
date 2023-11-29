package context

type Global struct {
	Job map[string]string
	Env map[string]string
}

func NewGlobal() *Global {
	return &Global{
		Job: map[string]string{},
		Env: map[string]string{},
	}
}

type Steps struct {
	Outputs map[string]map[string]string
}

func NewSteps() *Steps {
	return &Steps{
		Outputs: map[string]map[string]string{},
	}
}
