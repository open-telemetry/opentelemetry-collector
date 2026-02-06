// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfgen"

import (
	"os"
	"path/filepath"
)

const fileName = "config.schema.json"

// WriteJSONSchema writes the given ConfigMetadata as a JSON Schema file
// named "config.schema.json" in the specified directory.
func WriteJSONSchema(dir string, md *ConfigMetadata) error {
	filePath := filepath.Join(dir, fileName)
	data, err := md.ToJSON()
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0o600)
}
