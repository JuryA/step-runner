package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
)

type OutputValue struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type Outputs struct {
	outputFile string
}

func NewOutputs(outputFile string) *Outputs {
	return &Outputs{
		outputFile: outputFile,
	}
}

func (o *Outputs) Write(downloadDir string, imgRef name.Reference) error {
	writer, err := os.Create(o.outputFile)
	if err != nil {
		return fmt.Errorf("opening output file: %w", err)
	}
	defer writer.Close()

	outputValues := []OutputValue{
		{Name: "download_dir", Value: downloadDir},
		{Name: "ref", Value: imgRef.String()},
	}

	for _, outputValue := range outputValues {
		err := o.writeValue(outputValue, writer)
		if err != nil {
			return fmt.Errorf("value %s: %w", outputValue.Name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close output file: %w", err)
	}

	return nil
}

func (o *Outputs) writeValue(outputValue OutputValue, writer *os.File) error {
	jsonBytes, err := json.Marshal(outputValue)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	_, err = writer.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("write json to file: %w", err)
	}

	_, err = writer.WriteString("\n")
	if err != nil {
		return fmt.Errorf("write new line to file: %w", err)
	}

	return nil
}
