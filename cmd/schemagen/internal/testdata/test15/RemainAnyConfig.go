// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test15

type RemainAnyConfig struct {
	Name  string      `mapstructure:"name"`
	Extra interface{} `mapstructure:",remain"`
}
