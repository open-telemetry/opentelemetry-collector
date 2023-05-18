// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpprovider // import "go.opentelemetry.io/collector/confmap/provider/httpprovider"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"
)

// New returns a new confmap.Provider that reads the configuration from a http server.
//
// This Provider supports "http" scheme.
//
// One example for HTTP URI is: http://localhost:3333/getConfig
func New() confmap.Provider {
	return configurablehttpprovider.New(configurablehttpprovider.HTTPScheme)
}
