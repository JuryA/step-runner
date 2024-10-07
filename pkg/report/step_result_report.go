package report

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	protobuf "google.golang.org/protobuf/proto"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Format string

const (
	FormatJSON      Format = "json"
	FormatProtoText Format = "prototext"
)

type StepResultReport struct {
	filename string
	format   Format
}

func NewStepResultReport(filename string, format Format) *StepResultReport {
	return &StepResultReport{
		filename: filename,
		format:   format,
	}
}

func (r *StepResultReport) Write(result *proto.StepResult) error {

	var marshal func(protobuf.Message) ([]byte, error)
	switch r.format {
	case FormatJSON:
		marshal = protojson.Marshal
	case FormatProtoText:
		marshal = prototext.Marshal
	default:
		return fmt.Errorf("unsupported format: %v", r.format)
	}

	data, err := marshal(result)

	if err != nil {
		return fmt.Errorf("failed to write step results report: %w", err)
	}

	var filename string
	switch {
	case r.filename == "" && r.format == FormatJSON:
		filename = "step-results.json"
	case r.filename == "" && r.format == FormatProtoText:
		filename = "step-results.txtpb"
	default:
		filename = r.filename
	}

	err = os.WriteFile(filename, data, 0640)

	if err != nil {
		return fmt.Errorf("failed to write step results report: %w", err)
	}

	return nil
}
