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
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	destination = path.Join(destination, "step-runner")

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", source, err)
	}
	defer src.Close()

	dest, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	if err := dest.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return os.Chmod(destination, 0o755)
}
