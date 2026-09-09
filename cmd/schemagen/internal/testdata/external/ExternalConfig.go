// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package external

type TestConfig struct {
	Foo string `mapstructure:"foo"`
	Bar int    `mapstructure:"bar"`
}
