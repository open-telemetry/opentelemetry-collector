// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test04

import "go.opentelemetry.io/collector/cmd/schemagen/internal/testdata/test04/internal"

type RefFieldConfig struct {
	Database DatabaseConfig          `mapstructure:"database"`
	Network  NetworkConfig           `mapstructure:"network"`
	Auth     internal.Authentication `mapstructure:"auth"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
