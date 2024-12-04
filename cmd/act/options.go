package act

import "fmt"

type Options struct {
	WorkDir   *string
	ProtoStep *CLIProtoStep
	Env       *NameValues
	Job       *NameValues
}

func NewOptions() *Options {
	return &Options{
		ProtoStep: &CLIProtoStep{},
		Env:       &NameValues{},
		Job:       &NameValues{},
	}
}

func (o *Options) Validate() error {
	if o.WorkDir == nil || *o.WorkDir == "" {
		return fmt.Errorf("work dir is required")
	}

	if o.ProtoStep == nil {
		return fmt.Errorf("proto step is required")
	}

	if o.Env == nil {
		return fmt.Errorf("env is required")
	}

	if o.Job == nil {
		return fmt.Errorf("job is required")
	}

	return nil
}
