// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package external

type OtherConfig struct {
	Baz  int64      `mapstructure:"baz"`
	Test TestConfig `mapstructure:"test"`
}
