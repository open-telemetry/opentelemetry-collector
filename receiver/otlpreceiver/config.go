// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"encoding"
	"errors"
	"fmt"
	"net/url"
	"path"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
)

const (
	// Protocol values.
	protoGRPC = "protocols::grpc"
	protoHTTP = "protocols::http"
)

type SanitizedURLPath string

var _ encoding.TextUnmarshaler = (*SanitizedURLPath)(nil)

func (s *SanitizedURLPath) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err != nil {
		return fmt.Errorf("invalid HTTP URL path set for signal: %w", err)
	}

	if !path.IsAbs(u.Path) {
		u.Path = "/" + u.Path
	}

	*s = SanitizedURLPath(u.Path)
	return nil
}

type HTTPConfig struct {
	ServerConfig confighttp.ServerConfig `mapstructure:",squash"`

	// The URL path to receive traces on. If omitted "/v1/traces" will be used.
	TracesURLPath SanitizedURLPath `mapstructure:"traces_url_path,omitempty"`

	// The URL path to receive metrics on. If omitted "/v1/metrics" will be used.
	MetricsURLPath SanitizedURLPath `mapstructure:"metrics_url_path,omitempty"`

	// The URL path to receive logs on. If omitted "/v1/logs" will be used.
	LogsURLPath SanitizedURLPath `mapstructure:"logs_url_path,omitempty"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Protocols is the configuration for the supported protocols.
type Protocols struct {
	GRPC *configgrpc.ServerConfig `mapstructure:"grpc"`
	HTTP *HTTPConfig              `mapstructure:"http"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// Config defines configuration for OTLP receiver.
type Config struct {
	// Protocols is the configuration for the supported protocols, currently gRPC and HTTP (Proto and JSON).
	Protocols `mapstructure:"protocols"`
}

var (
	_ component.Config    = (*Config)(nil)
	_ confmap.Unmarshaler = (*Config)(nil)
)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.GRPC == nil && cfg.HTTP == nil {
		return errors.New("must specify at least one protocol when using the OTLP receiver")
	}
	return nil
}

// Unmarshal a confmap.Conf into the config struct.
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	// first load the config normally
	err := conf.Unmarshal(cfg)
	if err != nil {
		return err
	}

	if !conf.IsSet(protoGRPC) {
		cfg.GRPC = nil
	}

	if !conf.IsSet(protoHTTP) {
		cfg.HTTP = nil
	}

	return nil
}
