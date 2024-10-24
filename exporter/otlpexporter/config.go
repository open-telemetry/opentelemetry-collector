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
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for OTLP exporter.
type Config struct {
	exporterhelper.TimeoutConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
	RetryConfig                  configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// Experimental: This configuration is at the early stage of development and may change without backward compatibility
	// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved
	BatcherConfig exporterbatcher.Config `mapstructure:"batcher"`

	configgrpc.ClientConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
}

func (c *Config) Validate() error {
	endpoint := c.sanitizedEndpoint()
	if endpoint == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}
	hostport := endpoint
	if strings.HasPrefix(c.Endpoint, "http") {
		// see https://github.com/open-telemetry/opentelemetry-collector/issues/10488
		idx := strings.Index(hostport, "/")
		if idx > -1 {
			hostport = hostport[:idx]
		}
	} else if strings.HasPrefix(c.Endpoint, "dns") {
		// see https://github.com/open-telemetry/opentelemetry-collector/issues/10488
		validDNSRegex := regexp.MustCompile("^dns://([^/]+)/([^/]+)$|^dns:///([^/]+)$")
		if !validDNSRegex.Match([]byte(c.Endpoint)) {
			return fmt.Errorf("invalid dns scheme format")
		}
		matches := validDNSRegex.FindStringSubmatch(c.Endpoint)
		if len(matches) > 1 {
			if matches[2] != "" {
				// with authority
				hostport = matches[2]
			} else if matches[3] != "" {
				// without authority
				hostport = matches[3]
			}
		}
	}
	// Validate that the port is in the address
	_, port, err := net.SplitHostPort(hostport)
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
	case strings.HasPrefix(c.Endpoint, "http://"):
		return strings.TrimPrefix(c.Endpoint, "http://")
	case strings.HasPrefix(c.Endpoint, "https://"):
		return strings.TrimPrefix(c.Endpoint, "https://")
	case strings.HasPrefix(c.Endpoint, "dns://"):
		r := regexp.MustCompile("^dns://[/]?")
		return r.ReplaceAllString(c.Endpoint, "")
	default:
		return c.Endpoint
	}
}

var _ component.Config = (*Config)(nil)
