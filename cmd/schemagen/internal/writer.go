// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"errors"
	"os"
	"path/filepath"
)

func WriteSchemaToFile(schema *Schema, config *Config) (string, error) {
	schemaPath := createOutputFilePath(config)
	var (
		err error
		raw []byte
	)
	switch config.FileType {
	case "yaml", "yml":
		raw, err = schema.ToYAML()
		if err != nil {
			return "", err
		}
		raw, err = ApplyOverlayToYAML(raw, config)
	case "json":
		raw, err = schema.ToJSON()
		if err != nil {
			return "", err
		}
		raw, err = ApplyOverlayToJSON(raw, config)
	default:
		err = errors.New("unknown output file type; use json or yaml")
	}
	if err != nil {
		return "", err
	}

	err = writeFile(schemaPath, raw)
	if err != nil {
		return "", err
	}

	return schemaPath, nil
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func createOutputFilePath(config *Config) string {
	fileName := "config.schema." + config.FileType
	return filepath.Join(config.OutputFolder, fileName)
}
