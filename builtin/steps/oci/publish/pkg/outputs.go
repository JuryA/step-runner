package pkg

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"google.golang.org/protobuf/types/known/structpb"
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

func (o *Outputs) Write(imgRef name.Reference, image v1.ImageIndex) error {
	writer, err := os.Create(o.outputFile)
	if err != nil {
		return fmt.Errorf("opening output file: %w", err)
	}
	defer writer.Close()

	digest, err := image.Digest()
	if err != nil {
		return fmt.Errorf("getting digest of image index: %w", err)
	}

	digestOutput, err := structpb.NewStruct(map[string]any{
		"algorithm": digest.Algorithm,
		"hash":      digest.Hex,
		"value":     digest.String(),
	})
	if err != nil {
		return fmt.Errorf("creating digest output: %w", err)
	}

	outputValues := []OutputValue{
		{Name: "registry", Value: imgRef.Context().RegistryStr()},
		{Name: "repository", Value: imgRef.Context().RepositoryStr()},
		{Name: "tag", Value: imgRef.Identifier()},
		{Name: "ref", Value: imgRef.String()},
		{Name: "digest", Value: digestOutput},
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
