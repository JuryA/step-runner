package oci

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/releaser"
)

var Cmd = &cobra.Command{
	Use:   "oci",
	Short: "OCI Step Artifact management",
	Args:  cobra.ExactArgs(0),
}

func init() {
	loginCmd := &cobra.Command{
		Use:   "login <server>",
		Short: "Login to OCI repository",
		Args:  cobra.ExactArgs(1),
		RunE:  login,
	}

	loginCmd.Flags().StringP("username", "u", "", "Username for OCI repository")
	loginCmd.Flags().StringP("password", "p", "", "Password for OCI repository")
	loginCmd.Flags().Bool("password-stdin", false, "Take the password from stdin")

	Cmd.AddCommand(
		&cobra.Command{
			Use:   "release <repository>:version",
			Short: "Pack and push a component to an oci repository",
			Args:  cobra.ExactArgs(1),
			RunE:  release,
		},
		&cobra.Command{
			Use:   "list <repository>[:constraint]",
			Short: "List versions for repository",
			Args:  cobra.ExactArgs(1),
			RunE:  list,
		},
		loginCmd,
	)
}

func release(cmd *cobra.Command, args []string) error {
	name, err := releaser.Release(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	fmt.Println(name)

	return nil
}

func list(cmd *cobra.Command, args []string) error {
	versions, err := oci.List(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	for _, version := range versions {
		fmt.Println(version)
	}

	return nil
}

func login(cmd *cobra.Command, args []string) error {
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	if stdin, _ := cmd.Flags().GetBool("password-stdin"); stdin {
		pass, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("reading password from stdin:", err)
			os.Exit(1)
		}
		password = strings.TrimSuffix(strings.TrimSuffix(string(pass), "\n"), "\r")
	}

	via, err := oci.Login(args[0], username, password)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	fmt.Println("logged in via", via)

	return nil
}
