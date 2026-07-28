// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test00

type SimpleConfig struct {
	// Host is the server host
	Host string `mapstructure:"host"`
	// Port is the server port
	Port int `mapstructure:"port"`
	// Enabled indicates if the server is enabled
	Enabled bool `mapstructure:"enabled"`
	// Timeout in seconds
	Timeout float64 `mapstructure:"timeout"`
	// EnvVars is a map of environment variables
	EnvVars map[string]any `mapstructure:"env_vars"`

	_ struct{}
}
