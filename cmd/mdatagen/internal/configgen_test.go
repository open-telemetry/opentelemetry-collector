// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// An optional file which may be present in the test directory.
// It contains a mapping of the filenames which the code generator uses and the actual filenames on disk.
// If the file is not present, a default will be used:
//   - filename: generated_config.go
//     contents: "./generated_config.go"
var expectedFilesFilename = "expected_files.yaml"

// The contents of the "expected_files.yaml" file.
type ExpectedFilesSlice []ExpectedFiles

type ExpectedFiles struct {
	// The filename which the code generator uses.
	Filename string `yaml:"filename"`

	// The actual filename on disk, in the test directory.
	Contents string `yaml:"contents"`
}

func TestJsonSchema(t *testing.T) {
	testsDir := `./testdata/config_gen`
	rootSchemaFile := `metadata.yaml`

	tests, err := os.ReadDir(testsDir)
	require.NoError(t, err)

	for _, testDir := range tests {
		if !testDir.IsDir() {
			continue
		}

		t.Run(testDir.Name(), func(t *testing.T) {
			md, err := LoadMetadata(filepath.Join(testsDir, testDir.Name(), rootSchemaFile))
			require.NoError(t, err)

			expectedFilesPath := filepath.Join(testsDir, testDir.Name(), expectedFilesFilename)
			var expectedFiles []ExpectedFiles

			// Check if expected_files.yaml exists
			if _, err := os.Stat(expectedFilesPath); os.IsNotExist(err) {
				expectedFiles = []ExpectedFiles{
					{
						Filename: ConfigOutputFile,
						Contents: fmt.Sprintf("./%s", ConfigOutputFile),
					},
				}
			} else {
				expectedFilesData, err := os.ReadFile(filepath.Clean(expectedFilesPath))
				require.NoError(t, err)

				err = yaml.Unmarshal(expectedFilesData, &expectedFiles)
				require.NoError(t, err)
			}

			generatedConfigs, err := GenerateConfig("test_pkg", "test_dir", md.Config, md.ConfigSources)
			require.NoError(t, err)

			require.Len(t, generatedConfigs, 1+len(md.ConfigSources))

			for _, expectedFile := range expectedFiles {
				fileContents, err := findFile(expectedFile.Filename, generatedConfigs)
				require.NoError(t, err)

				expectedOutputFile := filepath.Join(testsDir, testDir.Name(), expectedFile.Contents)
				var expectedOutput []byte
				expectedOutput, err = os.ReadFile(filepath.Clean(expectedOutputFile))
				require.NoError(t, err)

				expectedOutputStr := normalizeLineEndings(string(expectedOutput))
				fileContents = normalizeLineEndings(fileContents)
				require.Equal(t, expectedOutputStr, fileContents)
			}
		})
	}
}

// normalizeLineEndings converts all line endings to Unix-style (\n)
// to ensure consistent comparison across different operating systems.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func filename(path string) string {
	_, filename := filepath.Split(path)
	return filename
}

func findFile(expectedFilename string, generatedConfigs map[string]string) (string, error) {
	for currentFilename, fileContents := range generatedConfigs {
		if expectedFilename == filename(currentFilename) {
			return string(fileContents), nil
		}
	}
	return "", fmt.Errorf("file %s not found in generated configs", expectedFilename)
}
