// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/atombender/go-jsonschema/pkg/generator"
	"github.com/atombender/go-jsonschema/pkg/schemas"
)

const (
	ConfigName = "config"
)

// GenerateConfig generates a "config.go", as well as any other Go files which "config.go" depends on.
// The inputs are:
// * "goPkgName" is the Go package at the top of the "config.go" file. For example, "batchprocessor".
// * "dir" is the location where the "config.go" file will be written. For example, "./processor/batchprocessor".
// * "conf" is the schema for "config.go". It is a "map[string]any".
//
// The output is a map, where:
// * The key is the absolute path to the file which must be written.
// * The value is the content of the file.
func GenerateConfig(goPkgName string, dir string, conf any) (map[string]string, error) {
	// load config
	jsonBytes, err := json.Marshal(conf)
	if err != nil {
		return nil, fmt.Errorf("failed loading config %w", err)
	}
	var schema schemas.Schema
	if err = json.Unmarshal(jsonBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	defaultOutputDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
	}
	defaultOutputNameDest := filepath.Join(defaultOutputDir, "config.go")

	// TODO: Make this configurable?
	repoRootDir := "../../"

	// TODO: Make this configurable. Or find a way to get rid of this mapping?
	schemaMappings := []generator.SchemaMapping{
		// {
		// 	SchemaID:    "opentelemetry.io/collector/exporter/exporterhelper/queue_sender",
		// 	PackageName: "go.opentelemetry.io/collector/exporter/exporterhelper",
		// 	OutputName:  "./exporter/exporterhelper/queue_sender.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configretry/backoff/retry_on_failure",
		// 	PackageName: "go.opentelemetry.io/collector/config/configretry",
		// 	OutputName:  "./config/configretry/backoff.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configtelemetry/configtelemetry",
		// 	PackageName: "go.opentelemetry.io/collector/config/configtelemetry",
		// 	OutputName:  "./config/configtelemetry/configtelemetry.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/confighttp/confighttp",
		// 	PackageName: "go.opentelemetry.io/collector/config/confighttp",
		// 	OutputName:  "./config/confighttp/confighttp.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/exporter/exporterhelper/timeout_sender",
		// 	PackageName: "go.opentelemetry.io/collector/exporter/exporterhelper",
		// 	OutputName:  "./exporter/exporterhelper/timeout_sender.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configtls/configtls",
		// 	PackageName: "go.opentelemetry.io/collector/config/configtls",
		// 	OutputName:  "./config/configtls/configtls.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configcompression/configcompression",
		// 	PackageName: "go.opentelemetry.io/collector/config/configcompression",
		// 	OutputName:  "./config/configcompression/compressiontype.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configauth/configauth",
		// 	PackageName: "go.opentelemetry.io/collector/config/configauth",
		// 	OutputName:  "./config/configauth/configauth.go",
		// },
		// {
		// 	SchemaID:    "opentelemetry.io/collector/config/configopaque",
		// 	PackageName: "go.opentelemetry.io/collector/config/configopaque",
		// 	OutputName:  "./config/configopaque/opaque.go",
		// },
	}
	for i := range schemaMappings {
		if schemaMappings[i].OutputName != "" {
			// The file paths in the schema mappings are relative to the repo root.
			// Make the paths absolute.
			relFilePath := filepath.Clean(filepath.Join(repoRootDir, schemaMappings[i].OutputName))
			absFilePath, errAbs := filepath.Abs(relFilePath)
			if errAbs != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", schemaMappings[i].OutputName, err)
			}
			absFilePath = filepath.Clean(absFilePath)
			schemaMappings[i].OutputName = absFilePath
		}
	}

	cfg := generator.Config{
		Warner: func(message string) {
			logf("Warning: %s", message)
		},
		DefaultPackageName:  goPkgName,
		DefaultOutputName:   defaultOutputNameDest,
		StructNameFromTitle: true,
		Tags:                []string{"mapstructure"},
		SchemaMappings:      schemaMappings,
		YAMLExtensions:      []string{".yaml", ".yml"},
		DisableOmitempty:    true,
	}

	generator, err := generator.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}
	if err = generator.AddFile(ConfigName, &schema); err != nil {
		return nil, fmt.Errorf("failed to add config: %w", err)
	}

	output := make(map[string]string)
	sources, err := generator.Sources()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Go files from schema: %w", err)
	}
	for sourceName, source := range sources {
		fmt.Printf("Writing to %s\n", sourceName)
		output[sourceName] = string(source)
	}
	fmt.Println("done")
	return output, nil
}

func logf(format string, args ...any) {
	fmt.Fprint(os.Stderr, "go-jsonschema: ")
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprint(os.Stderr, "\n")
}
