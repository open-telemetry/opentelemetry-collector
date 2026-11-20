// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

var _ component.Config = (*Config)(nil)

// validateConfig checks if the extension configuration is valid.
func validateConfig(cfg *Config) error {
	if cfg.ServerConfig.NetAddr.Endpoint == "" {
		return errors.New("\"endpoint\" is required when using the \"zpages\" extension")
	}
	return nil
}
