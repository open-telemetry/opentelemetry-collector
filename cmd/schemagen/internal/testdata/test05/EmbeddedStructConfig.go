// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test05

type EmbeddedStructConfig struct {
	DbConfig     `mapstructure:",squash"`
	AppName      string `mapstructure:"app_name"`
	internalInfo `mapstructure:",squash"`
}

type DbConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type internalInfo struct {
	Version string `mapstructure:"version"`
}
