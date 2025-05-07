// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for OTLP exporter.
type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig    `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig       `mapstructure:"retry_on_failure"`
	ClientConfig  configgrpc.ClientConfig         `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// Experimental: This configuration is at the early stage of development and may change without backward compatibility
	// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved
	//
	// Deprecated: [v0.123.0] batch configuration moving to queue configuration.
	BatcherConfig exporterhelper.BatcherConfig `mapstructure:"batcher"` //nolint:staticcheck // SA1019

	// remove at the same time as BatcherConfig
	hasBatcher bool
}

func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if conf.IsSet("batcher") {
		c.BatcherConfig = exporterhelper.NewDefaultBatcherConfig() //nolint:staticcheck // SA1019
		c.BatcherConfig.Enabled = false
		c.hasBatcher = true
	}

	if err := conf.Unmarshal(c); err != nil {
		return err
	}

	return nil
}

func (c *Config) Validate() error {
	endpoint := c.sanitizedEndpoint()
	if endpoint == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}

	// Validate that the port is in the address
	_, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf(`invalid port "%s"`, port)
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
		r := regexp.MustCompile("^dns://[/]?")
		return r.ReplaceAllString(c.ClientConfig.Endpoint, "")
	default:
		return c.ClientConfig.Endpoint
	}
}

var _ component.Config = (*Config)(nil)
