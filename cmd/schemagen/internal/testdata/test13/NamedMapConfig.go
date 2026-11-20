// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test13

// Headers is a named map of strings.
type Headers map[string]string

type Console map[string]interface{}

type NamedMapConfig struct {
	Headers Headers `mapstructure:"headers"`
	Console Console `mapstructure:"console"`
}
