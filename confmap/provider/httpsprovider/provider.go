// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpsprovider // import "go.opentelemetry.io/collector/confmap/provider/httpsprovider"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal/configurablehttpprovider"
)

// New returns a new confmap.Provider that reads the configuration from a https server.
//
// This Provider supports "https" scheme. One example of an HTTPS URI is: https://localhost:3333/getConfig
//
// To add extra CA certificates you need to install certificates in the system pool. This procedure is operating system
// dependent. E.g.: on Linux please refer to the `update-ca-trust` command.
func New() confmap.Provider {
	return configurablehttpprovider.New(configurablehttpprovider.HTTPSScheme)
}
