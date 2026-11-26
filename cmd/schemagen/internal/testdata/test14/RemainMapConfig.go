// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test14

type RemainMapConfig struct {
	Name  string            `mapstructure:"name"`
	Extra map[string]string `mapstructure:",remain"`
}
