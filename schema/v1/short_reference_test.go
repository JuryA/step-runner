package schema

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestShortReference_compileLocal(t *testing.T) {
	tests := []struct {
		name           string
		shortRef       shortReference
		expectPath     []string
		expectFilename string
	}{
		{
			name:           "using relative path",
			shortRef:       "./path/to/step.yml",
			expectPath:     []string{".", "path", "to"},
			expectFilename: "step.yml",
		},
		{
			name:           "using relative path without step.yml",
			shortRef:       "./path/to/step",
			expectPath:     []string{".", "path", "to", "step"},
			expectFilename: "step.yml",
		},
		{
			name:           "step.yml in current directory",
			shortRef:       "./step.yml",
			expectPath:     []string{"."},
			expectFilename: "step.yml",
		},
		{
			name:           "using absolute path",
			shortRef:       "/path/to/step.yml",
			expectPath:     []string{"/", "path", "to"},
			expectFilename: "step.yml",
		},
		{
			name:           "using absolute path without step.yml",
			shortRef:       "/path/to/step",
			expectPath:     []string{"/", "path", "to", "step"},
			expectFilename: "step.yml",
		},
		{
			name:           "using absolute path with slash suffix",
			shortRef:       "/path/to/step/",
			expectPath:     []string{"/", "path", "to", "step"},
			expectFilename: "step.yml",
		},
		{
			name:           "step.yml in root directory",
			shortRef:       "/step.yml",
			expectPath:     []string{"/"},
			expectFilename: "step.yml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			protoStepRef, err := test.shortRef.compile()
			require.NoError(t, err)
			require.Equal(t, protoStepRef.Protocol, proto.StepReferenceProtocol_local)

			// nolint:staticcheck // SA1019
			require.Equal(t, test.expectPath, protoStepRef.Path)
			require.Equal(t, test.expectFilename, protoStepRef.Filename)
		})
	}
}

func TestShortReference_compileDynamic(t *testing.T) {
	t.Run("entire ref is expression", func(t *testing.T) {
		var shortRef shortReference = "${{steps.previous_step.outputs.download_dir}}"

		protoStepRef, err := shortRef.compile()
		require.NoError(t, err)
		require.Equal(t, proto.StepReferenceProtocol_dynamic, protoStepRef.Protocol)
		require.Equal(t, "${{steps.previous_step.outputs.download_dir}}", protoStepRef.Url)
	})

	t.Run("ref is part expression", func(t *testing.T) {
		var shortRef shortReference = "/path/to/some/steps/${{env.THE_STEP}}"

		protoStepRef, err := shortRef.compile()
		require.NoError(t, err)
		require.Equal(t, proto.StepReferenceProtocol_dynamic, protoStepRef.Protocol)
		require.Equal(t, "/path/to/some/steps/${{env.THE_STEP}}", protoStepRef.Url)
	})
}
