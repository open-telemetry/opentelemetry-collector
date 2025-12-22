// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplereceiver // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
)

var _ component.Factory = (*AnotherStruct)(nil)

type AnotherStruct struct{}

func (a AnotherStruct) Type() component.Type {
	// TODO implement me
	panic("implement me")
}

func (a AnotherStruct) CreateDefaultConfig() component.Config {
	// TODO implement me
	panic("implement me")
}

var _ component.Config = (*MyConfig)(nil)

type CustomString string

// MyConfig defines configuration for the sample exporter used to test schema generation.
type MyConfig struct {
	// ID is the component identifier.
	ID component.ID `mapstructure:"id"`

	// Endpoint is the target URL to send data to.
	Endpoint string `mapstructure:"endpoint"`

	// CustomString is a custom string.
	CustomString CustomString `mapstructure:"custom_string"`

	// Timeout is the maximum time to wait for a response.
	Timeout time.Duration `mapstructure:"timeout"`

	// Enabled controls whether the exporter is active.
	Enabled bool `mapstructure:"enabled"`

	// BatchSize is the number of items to send in each batch.
	BatchSize int `mapstructure:"batch_size"`

	// Headers are additional headers to include in requests.
	Headers map[string]string `mapstructure:"headers"`

	// Retry contains retry configuration.
	Retry RetryConfig `mapstructure:"retry"`

	// Tags are optional tags to attach.
	Tags []string `mapstructure:"tags"`

	// APIKey is a secret API key (opaque string).
	APIKey configopaque.String `mapstructure:"api_key"`

	// OptionalRetry is an optional retry configuration.
	OptionalRetry configoptional.Optional[RetryConfig] `mapstructure:"optional_retry"`

	// Secrets is a list of secret key-value pairs.
	Secrets configopaque.MapList `mapstructure:"secrets"`

	// Endpoints is a list of endpoint configurations.
	Endpoints []EndpointConfig `mapstructure:"endpoints"`
}

// EndpointConfig holds configuration for a single endpoint.
type EndpointConfig struct {
	// URL is the endpoint URL.
	URL string `mapstructure:"url"`

	// Priority is the endpoint priority.
	Priority int `mapstructure:"priority"`
}

// RetryConfig holds retry settings.
type RetryConfig struct {
	// MaxRetries is the maximum number of retries.
	MaxRetries int `mapstructure:"max_retries"`

	// InitialInterval is the initial retry interval.
	InitialInterval time.Duration `mapstructure:"initial_interval"`
}
