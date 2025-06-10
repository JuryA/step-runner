package remote

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// ListTags lists the tags for the repository
// For example, for registry.gitlab.com/gitlab-org/gitlab-runner it will return v17.7.0, v17.7.1, v17.8.0, v17.8.1, etc
// No tags are returned if the repository is not found
func ListTags(ctx context.Context, repository name.Repository) ([]string, error) {
	slog.Debug("listing tags", "repository", repository.String())

	tags, err := remote.List(repository, remoteOptions(ctx)...)

	if err != nil {
		if isErrorNameUnknown(err) {
			slog.Debug("no tags, repository not found")
			return []string{}, nil
		}

		return nil, fmt.Errorf("listing tags: %w", err)
	}

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		for chunk := range slices.Chunk(tags, 20) {
			slog.Debug("tags", "tags", strings.Join(chunk, ","))
		}

		if len(tags) == 0 {
			slog.Debug("no tags")
		}
	}

	return tags, nil
}

func isErrorNameUnknown(err error) bool {
	for err != nil {
		var transportErr *transport.Error

		if errors.As(err, &transportErr) {
			for _, diagnostic := range transportErr.Errors {
				if diagnostic.Code == transport.NameUnknownErrorCode {
					return true
				}
			}
		}

		err = errors.Unwrap(err)
	}

	return false
}
