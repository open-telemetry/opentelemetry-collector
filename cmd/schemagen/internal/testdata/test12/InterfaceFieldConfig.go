// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test12

type InterfaceFieldConfig struct {
	Named  string      `mapstructure:"named"`
	Any    interface{} `mapstructure:"any"`
	Inline interface {
		Foo()
	} `mapstructure:"inline"`
	List []interface{} `mapstructure:"list"`
}
