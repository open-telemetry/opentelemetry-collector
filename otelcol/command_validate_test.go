// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
)

func TestValidateSubCommandNoConfig(t *testing.T) {
	cmd := newValidateSubCommand(CollectorSettings{Factories: nopFactories}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.ErrorContains(t, err, "at least one config flag must be provided")
}

func TestValidateSubCommandInvalidComponents(t *testing.T) {
	filePath := filepath.Join("testdata", "otelcol-invalid-components.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, filePath))
	})
	cmd := newValidateSubCommand(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{filePath},
			ProviderFactories: []confmap.ProviderFactory{fileProvider},
			DefaultScheme:     "file",
		},
	}}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.ErrorContains(t, err, "unknown type: \"nosuchprocessor\"")
}
