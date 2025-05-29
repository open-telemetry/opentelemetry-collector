// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tinybirdexporter

import (
	"fmt"
	"net/url"

	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for the Tinybird exporter.
type Config struct {
	Endpoint   string `mapstructure:"endpoint"`
	Token      string `mapstructure:"token"`
	DataSource string `mapstructure:"datasource"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Token == "" {
		return errMissingToken
	}
	if cfg.DataSource == "" {
		return errMissingDataSource
	}
	if cfg.Endpoint == "" {
		return errMissingEndpoint
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("endpoint must be a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("endpoint must have http or https scheme: %s", cfg.Endpoint)
	}
	if u.Host == "" {
		return fmt.Errorf("endpoint must have a host: %s", cfg.Endpoint)
	}
	return nil
}
