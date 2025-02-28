package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOCIStepResource_NamedReference(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		tag        string
		expect     string
		expectErr  string
	}{
		{
			name:       "registry, repository, and tag",
			registry:   "registry.gitlab.com",
			repository: "group/project",
			tag:        "1.0.0",
			expect:     "registry.gitlab.com/group/project:1.0.0",
		},
		{
			name:       "removes extra slashes",
			registry:   "registry.gitlab.com//",
			repository: "/group/project/",
			tag:        "latest",
			expect:     "registry.gitlab.com/group/project:latest",
		},
		{
			name:       "registry with port",
			registry:   "registry.gitlab.com:8080",
			repository: "project",
			tag:        "latest",
			expect:     "registry.gitlab.com:8080/project:latest",
		},
		{
			name:       "invalid registry",
			registry:   "registry.gitlab.com/!",
			repository: "project",
			tag:        "latest",
			expectErr:  "could not parse reference: registry.gitlab.com/!/project:latest",
		},
		{
			name:       "invalid tag",
			registry:   "registry.gitlab.com",
			repository: "project",
			tag:        "!err!",
			expectErr:  "could not parse reference: registry.gitlab.com/project:!err!",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := NewOCIStepResource(test.registry, test.repository, test.tag, nil, "step.yml")
			reference, err := resource.NamedReference()
			if test.expectErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.expect, reference.String())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
			}
		})
	}
}
