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
	Global *Global
	Env map[string]string
	Outputs map[string]map[string]string
}

func NewSteps() *Steps {
	return &Steps{
		Outputs: map[string]map[string]string{},
	}
}

func (s *Steps) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.Global.Env {
		r[k] = v
	}
	for k, v := range s.Env {
		r[k] = v
	}
	return r
}

func (s *Steps) GetEnvList() []string {
	r := []string{};
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}

