package main

import (
	"os"
	"path/filepath"
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

		md, err := loadMetadata(filepath.Join(inputDir, inputFile.Name()))
		require.NoError(t, err)

		generatedConfigs, err := GenerateConfig("test_pkg", "test_dir", md.Config)
		require.NoError(t, err)

		expectedOutputFile := filepath.Join(outputDir, inputFile.Name())
		var expectedOutput []byte
		expectedOutput, err = os.ReadFile(expectedOutputFile)
		require.NoError(t, err)

		require.Equal(t, 1, len(generatedConfigs))

		for _, fileContents := range generatedConfigs {
			require.Equal(t, string(expectedOutput), fileContents)
			//TODO: Also check the filename (the key in the map).
		}
	}
}
