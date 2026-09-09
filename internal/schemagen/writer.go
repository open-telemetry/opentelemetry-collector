// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const fileName = "config.schema.json"

// WriteJSONSchema writes the given ConfigMetadata as a JSON Schema file
// named "config.schema.json" in the specified directory.
func WriteJSONSchema(dir, id, title string, md *ConfigsMetadata) error {
	filePath := filepath.Join(dir, fileName)

	jsonSchema := FromMetadata(id, title, md)

	data, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0o600)
}
