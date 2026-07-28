// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test02

type Config struct {
	Connection struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"connection"`
}
