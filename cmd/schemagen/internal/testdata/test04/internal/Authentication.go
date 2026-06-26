// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

type Authentication struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
