package bootstrap

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap <destination>",
		Short: "Copy the step-runner binary to the destination path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get source path: %w", err)
			}

			return run(source, args[0])
		},
	}
}

func run(source, destination string) error {
	info, err := os.Stat(destination)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("destination %q is not a directory", destination)
	}

	destination = path.Join(destination, "step-runner")

	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination %q already exists", destination)
	}

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", source, err)
	}
	defer src.Close()

	dest, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", destination, err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return dest.Close()
}
