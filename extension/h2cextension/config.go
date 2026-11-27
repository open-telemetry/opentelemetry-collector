// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package h2cextension // import "go.opentelemetry.io/collector/extension/h2cextension"

import (
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

// Config is the configuration for the h2c extension.
type Config struct {
	_ struct{}
}

var (
	_ component.Config                             = (*Config)(nil)
	_ extensionmiddleware.HTTPServerProtocolConfig = (*Config)(nil)
)

// RequiredHTTPServerProtocols declares the protocol the listener must accept.
func (*Config) RequiredHTTPServerProtocols() http.Protocols {
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	return protocols
}

// Validate checks the extension configuration.
func (*Config) Validate() error {
	return nil
}
