// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

type Ignore struct {
	Top []string `mapstructure:"top"`
	Any []string `mapstructure:"any"`
}

type GoLeak struct {
	Skip     bool   `mapstructure:"skip"`
	Ignore   Ignore `mapstructure:"ignore"`
	Setup    string `mapstructure:"setup"`
	Teardown string `mapstructure:"teardown"`
}

type InvalidConfig struct {
	Name   string         `mapstructure:"name"`
	Config map[string]any `mapstructure:"config"`
	Error  string         `mapstructure:"error"`
}

type Tests struct {
	Config              any             `mapstructure:"config"`
	InvalidConfigs      []InvalidConfig `mapstructure:"invalid_configs"`
	SkipLifecycle       bool            `mapstructure:"skip_lifecycle"`
	SkipShutdown        bool            `mapstructure:"skip_shutdown"`
	GoLeak              GoLeak          `mapstructure:"goleak"`
	ExpectConsumerError bool            `mapstructure:"expect_consumer_error"`
	Host                string          `mapstructure:"host"`
}
