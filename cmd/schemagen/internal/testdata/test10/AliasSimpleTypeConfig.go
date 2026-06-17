// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test10

type StringsArray []string

type Double float64

type AliasSimpleTypeConfig struct {
	Strings StringsArray `mapstructure:"list_of_strings"`
	Number  Double       `mapstructure:"num"`
}
