// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main // import "go.opentelemetry.io/collector/cmd/schemagen"

import (
	"fmt"
	"log"

	"go.opentelemetry.io/collector/cmd/schemagen/internal"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := internal.ReadConfig()
	if err != nil {
		return err
	}

	parser := internal.NewParser(config)
	schema, err := parser.Parse()
	if err != nil {
		return err
	}

	if config.ResolveRefs {
		resolver := internal.NewRefResolver(config)
		schema, err = resolver.Resolve(schema)
		if err != nil {
			return err
		}
	}

	path, writeErr := internal.WriteSchemaToFile(schema, config)
	if writeErr != nil {
		return writeErr
	}
	fmt.Println("Schema successfully written to", path)
	return nil
}
