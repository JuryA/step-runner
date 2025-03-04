package oci_test

import (
	"io/fs"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestArtifact_FS(t *testing.T) {
	t.Run("from is a file", func(t *testing.T) {
		t.Run("writes files", func(t *testing.T) {
			baseDir := bldr.Files(t).WriteFile("cow", "moo").Build()
			cowFile := filepath.Join(baseDir, "cow")

			tests := []struct {
				name       string
				from       string
				to         string
				resultFile string
			}{
				{
					name:       "writes to root folder",
					from:       cowFile,
					to:         "cow",
					resultFile: "cow",
				},
				{
					name:       "writes with additional slash in from path",
					from:       cowFile + "/",
					to:         "cow",
					resultFile: "cow",
				},
				{
					name:       "cleans paths",
					from:       cowFile + "/directory/..",
					to:         "directory/../cow",
					resultFile: "cow",
				},
				{
					name:       "creates new directories",
					from:       cowFile,
					to:         "my/animals/cow",
					resultFile: "my/animals/cow",
				},
				{
					name:       "writes when to path has slash prefix",
					from:       cowFile,
					to:         "/cow",
					resultFile: "cow",
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					artifact := oci.NewArtifact(bldr.OCIPlatform.Generic, test.from, test.to)

					fsys, cleanup, err := artifact.FS()
					require.NoError(t, err)
					t.Cleanup(func() { _ = cleanup() })

					data, err := fs.ReadFile(fsys, test.resultFile)
					require.NoError(t, err)
					require.Equal(t, "moo", string(data))
				})
			}
		})
	})

	t.Run("from is a directory", func(t *testing.T) {
		t.Run("writes files", func(t *testing.T) {
			baseDir := bldr.Files(t).WriteFile("animals/snake", "hiss").Build()
			artifact := oci.NewArtifact(bldr.OCIPlatform.Generic, baseDir, "/my_files")

			fsys, cleanup, err := artifact.FS()
			require.NoError(t, err)
			t.Cleanup(func() { _ = cleanup() })

			data, err := fs.ReadFile(fsys, "my_files/animals/snake")
			require.NoError(t, err)
			require.Equal(t, "hiss", string(data))
		})

		t.Run("writes directory and files", func(t *testing.T) {
			baseDir := bldr.Files(t).
				TouchFile("animals/snake").
				TouchFile("animals/birds/emu").
				Build()

			tests := []struct {
				name   string
				from   string
				to     string
				expect []string
			}{
				{
					name: "writes to new path",
					from: baseDir,
					to:   "creatures/real",
					expect: []string{
						"creatures",
						"creatures/real",
						"creatures/real/animals",
						"creatures/real/animals/birds",
						"creatures/real/animals/birds/emu",
						"creatures/real/animals/snake",
					},
				},
				{
					name: "slashes in to path are ignored",
					from: baseDir,
					to:   "/files/",
					expect: []string{
						"files",
						"files/animals",
						"files/animals/birds",
						"files/animals/birds/emu",
						"files/animals/snake",
					},
				},
				{
					name: "cleans paths",
					from: baseDir + "/directory/..",
					to:   "/directory/..",
					expect: []string{
						"animals",
						"animals/birds",
						"animals/birds/emu",
						"animals/snake",
					}},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					artifact := oci.NewArtifact(bldr.OCIPlatform.Generic, test.from, test.to)
					paths := make([]string, 0)

					fsys, cleanup, err := artifact.FS()
					require.NoError(t, err)
					t.Cleanup(func() { _ = cleanup() })

					err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
						if path != "." {
							paths = append(paths, path)
						}
						return err
					})
					require.NoError(t, err)

					sort.Strings(paths)
					require.Equal(t, test.expect, paths)
				})
			}
		})
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name      string
			from      string
			to        string
			expectErr string
		}{
			{
				name:      "from path does not exist",
				from:      "/file/doesnt/exist",
				expectErr: `stat /file/doesnt/exist: no such file or directory`,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				artifact := oci.NewArtifact(bldr.OCIPlatform.Generic, test.from, test.to)

				_, _, err := artifact.FS()
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
			})
		}
	})
}
