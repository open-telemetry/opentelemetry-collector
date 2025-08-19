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
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for OTLP exporter.
type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig    `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig       `mapstructure:"retry_on_failure"`
	ClientConfig  configgrpc.ClientConfig         `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// prevent unkeyed literal initialization
	_ struct{}
}

type EndpointValidator interface {
	Match(endpoint string) bool
	Sanitize(endpoint string) string
	Validate(endpoint string) error
}

type Pipeline struct {
	validators []EndpointValidator
}

func (p *Pipeline) Add(v EndpointValidator) {
	p.validators = append(p.validators, v)
}

func (p *Pipeline) Run(endpoint string) error {
	for _, v := range p.validators {
		if v.Match(endpoint) {
			sanitized := v.Sanitize(endpoint)
			return v.Validate(sanitized)
		}
	}
	return nil
}

type HostPortValidator struct {
	prefix string
}

func (v HostPortValidator) Match(ep string) bool {
	return v.prefix == "" || strings.HasPrefix(ep, v.prefix)
}

func (v HostPortValidator) Sanitize(ep string) string {
	if v.prefix == "" {
		return ep
	}
	if strings.HasPrefix(ep, "dns://") {
		r := regexp.MustCompile(`^dns:///?`)
		return r.ReplaceAllString(ep, "")
	}
	return strings.TrimPrefix(ep, v.prefix)
}

func (v HostPortValidator) Validate(ep string) error {
	if ep == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}

	_, port, err := net.SplitHostPort(ep)
	if err != nil {
		return err
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf(`invalid port "%s"`, port)
	}
	return nil
}

type UnixValidator struct{}

func (v UnixValidator) Match(ep string) bool {
	return strings.HasPrefix(ep, "unix:")
}

func (v UnixValidator) Sanitize(ep string) string {
	return ep
}

func (v UnixValidator) Validate(ep string) error {
	if strings.HasPrefix(ep, "unix://") {
		path := strings.TrimPrefix(ep, "unix://")
		if path == "" {
			return fmt.Errorf("empty unix socket path")
		}
		return nil
	} else if strings.HasPrefix(ep, "unix:@") {
		path := strings.TrimPrefix(ep, "unix:@")
		if path == "" {
			return fmt.Errorf("empty abstract unix socket path")
		}
		return nil
	}

	return fmt.Errorf("invalid unix socket protocol: %s", ep)
}

func (c *Config) Validate() error {
	pipeline := Pipeline{}
	pipeline.Add(HostPortValidator{prefix: "http://"})
	pipeline.Add(HostPortValidator{prefix: "https://"})
	pipeline.Add(HostPortValidator{prefix: "dns://"})
	pipeline.Add(UnixValidator{})
	pipeline.Add(HostPortValidator{prefix: ""}) // (host:port)

	return pipeline.Run(c.ClientConfig.Endpoint)
}

var _ component.Config = (*Config)(nil)
