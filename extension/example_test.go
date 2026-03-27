// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extension_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// exampleExtension is a minimal extension that reports its lifecycle.
type exampleExtension struct {
	id component.ID
}

// Start is called by the Collector when the extension is started.
func (e *exampleExtension) Start(_ context.Context, _ component.Host) error {
	fmt.Printf("extension %q started\n", e.id)
	return nil
}

// Shutdown is called by the Collector when the extension is stopped.
func (e *exampleExtension) Shutdown(_ context.Context) error {
	fmt.Printf("extension %q stopped\n", e.id)
	return nil
}

func createExampleConfig() component.Config {
	return struct{}{}
}

func createExampleExtension(_ context.Context, set extension.Settings, _ component.Config) (extension.Extension, error) {
	return &exampleExtension{id: set.ID}, nil
}

// Example demonstrates how a Factory creates an extension using Settings.ID.
func Example() {
	extType := component.MustNewType("example")

	factory := extension.NewFactory(
		extType,
		createExampleConfig,
		createExampleExtension,
		component.StabilityLevelDevelopment,
	)

	settings := extension.Settings{
		ID: component.NewID(extType),
	}

	ext, err := factory.Create(context.Background(), settings, factory.CreateDefaultConfig())
	if err != nil {
		panic(err)
	}

	_ = ext.Start(context.Background(), nil)
	_ = ext.Shutdown(context.Background())

	// Output:
	// extension "example" started
	// extension "example" stopped
}
