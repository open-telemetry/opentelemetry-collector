// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// ConnectionPoolConfig configures the gRPC connection pool.
// When present, connection pooling is enabled with the specified settings.
// When absent, a single connection is used (default behavior).
type ConnectionPoolConfig struct {
	// Size specifies the number of connections in the pool.
	// Must be greater than 0.
	// Default: 4
	Size int `mapstructure:"size"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Config defines configuration for OTLP exporter.
type Config struct {
	TimeoutConfig    exporterhelper.TimeoutConfig                             `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig      configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig      configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`
	ClientConfig     configgrpc.ClientConfig                                  `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	ConnectionPool   configoptional.Optional[ConnectionPoolConfig]            `mapstructure:"connection_pool"`

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

	// Validate connection pool configuration
	if c.ConnectionPool.HasValue() {
		poolCfg := c.ConnectionPool.Get()
		if poolCfg.Size <= 0 {
			return fmt.Errorf("connection_pool.size must be greater than 0, got %d", poolCfg.Size)
		}
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
