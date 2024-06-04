// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/featuregate"
)

func TestValidateSubCommandNoConfig(t *testing.T) {
	cmd := newValidateSubCommand(CollectorSettings{Factories: nopFactories}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one config flag must be provided")
}

func TestValidateSubCommandInvalidComponents(t *testing.T) {
	cmd := newValidateSubCommand(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:               []string{filepath.Join("testdata", "otelcol-invalid-components.yaml")},
			ProviderFactories:  []confmap.ProviderFactory{fileprovider.NewFactory()},
			ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
		},
	}}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown type: \"nosuchprocessor\"")
}
