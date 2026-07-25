// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test07

type ComplexTypeFieldConfig struct {
	ServiceName string                   `mapstructure:"service_name"`
	Enabled     bool                     `mapstructure:"enabled"`
	Endpoints   []Endpoint               `mapstructure:"endpoints"`
	Metadata    map[string]MetadataEntry `mapstructure:"metadata"`
	Routes      []map[string][]string    `mapstructure:"routes"`
	Credentials Credentials              `mapstructure:"credentials"`
	TLS         *TLSConfig               `mapstructure:"tls"`
	Retry       RetryPolicy              `mapstructure:"retry"`
	Inline      struct {
		Labels  map[string][]string `mapstructure:"labels"`
		Options []struct {
			Key   string `mapstructure:"key"`
			Value string `mapstructure:"value"`
		} `mapstructure:"options"`
	} `mapstructure:"inline"`
}

type Endpoint struct {
	Name    string              `mapstructure:"name"`
	URL     string              `mapstructure:"url"`
	Headers map[string][]string `mapstructure:"headers"`
}

type MetadataEntry struct {
	Value      string            `mapstructure:"value"`
	Attributes map[string]string `mapstructure:"attributes"`
}

type Credentials struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type TLSConfig struct {
	Insecure     bool          `mapstructure:"insecure"`
	Certificates []Certificate `mapstructure:"certificates"`
}

type Certificate struct {
	Path     string  `mapstructure:"path"`
	Password *string `mapstructure:"password"`
}

type RetryPolicy struct {
	MaxAttempts    int   `mapstructure:"max_attempts"`
	BackoffSeconds []int `mapstructure:"backoff_seconds"`
}
