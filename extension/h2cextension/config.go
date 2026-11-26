// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package h2cextension // import "go.opentelemetry.io/collector/extension/h2cextension"

import "go.opentelemetry.io/collector/component"

// Config is the configuration for the h2c extension.
type Config struct {
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks the extension configuration.
func (*Config) Validate() error {
	return nil
}
