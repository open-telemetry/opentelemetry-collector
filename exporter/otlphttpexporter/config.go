// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter // import "go.opentelemetry.io/collector/exporter/otlphttpexporter"

import (
	"encoding"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// EncodingType defines the type for content encoding
type EncodingType string

const (
	EncodingProto EncodingType = "proto"
	EncodingJSON  EncodingType = "json"
)

var _ encoding.TextUnmarshaler = (*EncodingType)(nil)

// UnmarshalText unmarshalls text to an EncodingType.
func (e *EncodingType) UnmarshalText(text []byte) error {
	if e == nil {
		return errors.New("cannot unmarshal to a nil *EncodingType")
	}

	str := string(text)
	switch str {
	case string(EncodingProto):
		*e = EncodingProto
	case string(EncodingJSON):
		*e = EncodingJSON
	default:
		return fmt.Errorf("invalid encoding type: %s", str)
	}

	return nil
}

// RetryConfig extends the standard BackOffConfig with additional HTTP-specific options.
type RetryConfig struct {
	configretry.BackOffConfig `mapstructure:",squash"`

	// RetryableStatuses is a list of HTTP status codes that should trigger retries.
	// By default, this is set to [429, 502, 503, 504] per OTLP spec.
	// Modify this list to control which codes are retried - codes not in this list
	// will be treated as permanent errors.
	RetryableStatuses []int `mapstructure:"retryable_statuses"`
}

// Config defines configuration for OTLP/HTTP exporter.
type Config struct {
	ClientConfig confighttp.ClientConfig                                  `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig  configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig  RetryConfig                                              `mapstructure:"retry_on_failure"`

	// The URL to send traces to. If omitted the Endpoint + "/v1/traces" will be used.
	TracesEndpoint string `mapstructure:"traces_endpoint"`

	// The URL to send metrics to. If omitted the Endpoint + "/v1/metrics" will be used.
	MetricsEndpoint string `mapstructure:"metrics_endpoint"`

	// The URL to send logs to. If omitted the Endpoint + "/v1/logs" will be used.
	LogsEndpoint string `mapstructure:"logs_endpoint"`

	// The URL to send profiles to. If omitted the Endpoint + "/v1development/profiles" will be used.
	ProfilesEndpoint string `mapstructure:"profiles_endpoint"`

	// The encoding to export telemetry (default: "proto")
	Encoding EncodingType `mapstructure:"encoding"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.ClientConfig.Endpoint == "" && cfg.TracesEndpoint == "" && cfg.MetricsEndpoint == "" && cfg.LogsEndpoint == "" && cfg.ProfilesEndpoint == "" {
		return errors.New("at least one endpoint must be specified")
	}

	for _, code := range cfg.RetryConfig.RetryableStatuses {
		if code < 100 || code > 599 {
			return fmt.Errorf("invalid HTTP status code in retry_on_failure.retryable_statuses: %d (must be in range 100-599)", code)
		}
	}

	return nil
}
