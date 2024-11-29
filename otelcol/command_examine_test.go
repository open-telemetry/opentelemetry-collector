// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/featuregate"
)

func TestExamineCommand(t *testing.T) {

	tests := []struct {
		name      string
		set       confmap.ResolverSettings
		errString string
	}{
		{
			name:      "no URIs",
			set:       confmap.ResolverSettings{},
			errString: "at least one config flag must be provided",
		},
		{
			name: "valid URI - file not found",
			set: confmap.ResolverSettings{
				URIs: []string{"file:blabla.yaml"},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
				DefaultScheme: "file",
			},
			errString: "cannot retrieve the configuration: unable to read the file",
		},
		{
			name: "valid URI",
			set: confmap.ResolverSettings{
				URIs: []string{"yaml:processors::batch/foo::timeout: 3s"},
				ProviderFactories: []confmap.ProviderFactory{
					yamlprovider.NewFactory(),
				},
				DefaultScheme: "yaml",
			},
		},
		{
			name: "valid URI - no provider set",
			set: confmap.ResolverSettings{
				URIs:          []string{"yaml:processors::batch/foo::timeout: 3s"},
				DefaultScheme: "yaml",
			},
			errString: "at least one Provider must be supplied",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := ConfigProviderSettings{
				ResolverSettings: test.set,
			}

			cmd := newExamineSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
			err := cmd.Execute()
			if test.errString != "" {
				require.ErrorContains(t, err, test.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
