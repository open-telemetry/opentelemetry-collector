// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test06

type PointerFieldConfig struct {
	Name           *string    `mapstructure:"name"`
	Tags           *[]string  `mapstructure:"tags"`
	Address        *Address   `mapstructure:"address"`
	OtherAddresses *[]Address `mapstructure:"other_addresses"`
}

type Address struct {
	Hostname string `mapstructure:"hostname"`
	Port     *int   `mapstructure:"port"`
	Path     string `mapstructure:"path"`
}
