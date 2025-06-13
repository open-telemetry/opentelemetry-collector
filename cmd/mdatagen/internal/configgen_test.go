// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonSchema(t *testing.T) {
	inputDir := `./testdata/config_gen/input_schema`
	outputDir := `./testdata/config_gen/expected_golang_output/`

	inputFiles, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	for _, inputFile := range inputFiles {
		if inputFile.IsDir() {
			continue
		}

		md, err := LoadMetadata(filepath.Join(inputDir, inputFile.Name()))
		require.NoError(t, err)

		generatedConfigs, err := GenerateConfig("test_pkg", "test_dir", md.Config)
		require.NoError(t, err)

		expectedOutputFile := filepath.Join(outputDir, inputFile.Name())
		var expectedOutput []byte
		expectedOutput, err = os.ReadFile(expectedOutputFile)
		require.NoError(t, err)

		require.Equal(t, 1, len(generatedConfigs))

		for filePathStr, fileContents := range generatedConfigs {
			require.Equal(t, "config.go", filePathLeaf(filePathStr))
			require.Equal(t, string(expectedOutput), fileContents)
		}
	}
}

func filePathLeaf(path string) string {
	split := strings.Split(path, "/")
	return split[len(split)-1]
}

func TestFilePathLeaf(t *testing.T) {
	input := "/Users/user/git/opentelemetry-collector/cmd/mdatagen/internal/test_dir/config.go"
	expected := "config.go"
	actual := filePathLeaf(input)
	require.Equal(t, expected, actual)
}
