// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package testcomponent provides a test configuration for schema generation testing.
// It covers all basic Go types, special types (time.Duration), nested structs,
// slices, maps, and deprecation detection.
// The config type is intentionally named "MySettings" (not "Config") to test
// that schema generation detects config types via compile-time checks.
package testcomponent // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent"

import (
	"context"
	"time"

	comp "go.opentelemetry.io/collector/component"
)

var _ comp.Component = (*OtherStruct)(nil)

type OtherStruct struct{}

func (o OtherStruct) Start(ctx context.Context, host comp.Host) error {
	//TODO implement me
	panic("implement me")
}

func (o OtherStruct) Shutdown(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// Compile-time check that MySettings implements component.Config.
var _ comp.Config = (*MySettings)(nil)

// MySettings is the configuration for the test component.
type MySettings struct {
	// Name is the name of the component.
	Name string `mapstructure:"name"`

	// Count is a numeric counter.
	Count int `mapstructure:"count"`

	// Enabled controls whether the component is enabled.
	Enabled bool `mapstructure:"enabled"`

	// Rate is a floating point rate.
	Rate float64 `mapstructure:"rate"`

	// Timeout is the timeout duration.
	Timeout time.Duration `mapstructure:"timeout"`

	// Tags is a list of tags.
	Tags []string `mapstructure:"tags"`

	// Numbers is a list of integers.
	Numbers []int `mapstructure:"numbers"`

	// Metadata is a key-value map.
	Metadata map[string]string `mapstructure:"metadata"`

	// Nested contains nested configuration.
	Nested NestedConfig `mapstructure:"nested"`

	// NestedPtr is a pointer to nested configuration.
	NestedPtr *NestedConfig `mapstructure:"nested_ptr"`

	// OldField is deprecated and should not be used.
	OldField string `mapstructure:"old_field"`
}

// NestedConfig is a nested configuration struct.
type NestedConfig struct {
	// Host is the server host.
	Host string `mapstructure:"host"`

	// Port is the server port.
	Port int `mapstructure:"port"`

	// DeepNested contains deeply nested configuration.
	DeepNested DeepNestedConfig `mapstructure:"deep_nested"`
}

// DeepNestedConfig is a deeply nested configuration struct.
type DeepNestedConfig struct {
	// Value is a configuration value.
	Value string `mapstructure:"value"`
}
