// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpprovider // import "go.opentelemetry.io/collector/confmap/provider/httpprovider"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"
)

// NewWithSettings returns a new confmap.Provider that reads the configuration from a http server.
//
// This Provider supports "http" scheme.
//
// One example for HTTP URI is: http://localhost:3333/getConfig
//
// Deprecated: [v0.99.0] Use NewFactory instead.
func NewWithSettings(set confmap.ProviderSettings) confmap.Provider {
	return configurablehttpprovider.New(configurablehttpprovider.HTTPScheme, set)
}

// NewFactory returns a factory for a confmap.Provider that reads the configuration from a http server.
//
// This Provider supports "http" scheme.
//
// One example for HTTP URI is: http://localhost:3333/getConfig
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(NewWithSettings)
}
