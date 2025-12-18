// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"errors"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for OTLP exporter.
type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig                             `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`
	ClientConfig  configgrpc.ClientConfig                                  `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// prevent unkeyed literal initialization
	_ struct{}
}

var (
	_ component.Config   = (*Config)(nil)
	_ xconfmap.Validator = (*Config)(nil)
)

func (c *Config) Validate() error {
	if endpoint := c.sanitizedEndpoint(); endpoint == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}
	return nil
}

func (c *Config) sanitizedEndpoint() string {
	switch {
	case strings.HasPrefix(c.ClientConfig.Endpoint, "http://"):
		return strings.TrimPrefix(c.ClientConfig.Endpoint, "http://")
	case strings.HasPrefix(c.ClientConfig.Endpoint, "https://"):
		return strings.TrimPrefix(c.ClientConfig.Endpoint, "https://")
	case strings.HasPrefix(c.ClientConfig.Endpoint, "dns://"):
		r := regexp.MustCompile(`^dns:///?`)
		return r.ReplaceAllString(c.ClientConfig.Endpoint, "")
	default:
		return c.ClientConfig.Endpoint
	}
}
