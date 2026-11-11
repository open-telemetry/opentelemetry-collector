// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter // import "go.opentelemetry.io/collector/exporter/otlphttpexporter"

import (
	"encoding"
	"errors"
	"fmt"
	"net/http"

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

// Config defines configuration for OTLP/HTTP exporter.
type Config struct {
	ClientConfig confighttp.ClientConfig                                  `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig  configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig  configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`

	// Additional HTTP status codes (401-599) to treat as retryable alongside
	// the OTLP defaults (429, 502, 503, 504). Listed codes also honor
	// Retry-After headers when present and require retry_on_failure to be enabled.
	RetryableHTTPStatusCodes []int `mapstructure:"retryable_http_status_codes"`

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
	if len(cfg.RetryableHTTPStatusCodes) > 0 && !cfg.RetryConfig.Enabled {
		return errors.New("retryable_http_status_codes cannot be configured when retry_on_failure is disabled")
	}
	seen := make(map[int]struct{}, len(cfg.RetryableHTTPStatusCodes))
	for _, code := range cfg.RetryableHTTPStatusCodes {
		if code == http.StatusBadRequest {
			return errors.New("retryable_http_status_codes cannot contain 400 (Bad Request)")
		}
		if code < 401 || code > 599 {
			return fmt.Errorf("retryable_http_status_codes must only contain HTTP status codes in the range 401-599, got %d", code)
		}
		if _, ok := seen[code]; ok {
			return fmt.Errorf("duplicate status code %d in retryable_http_status_codes", code)
		}
		seen[code] = struct{}{}
	}
	return nil
}
