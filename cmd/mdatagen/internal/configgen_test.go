// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type ExpectedFiles struct {
	Filename string `yaml:"filename"`
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

			expectedFilesPath := filepath.Join(testsDir, testDir.Name(), "expected_files.yaml")
			var expectedFiles []ExpectedFiles

			// Check if expected_files.yaml exists
			if _, err := os.Stat(expectedFilesPath); os.IsNotExist(err) {
				expectedFiles = []ExpectedFiles{
					{
						Filename: "config.go",
						Contents: "./config.go",
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

				require.Equal(t, string(expectedOutput), fileContents)
			}
		})
	}
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
