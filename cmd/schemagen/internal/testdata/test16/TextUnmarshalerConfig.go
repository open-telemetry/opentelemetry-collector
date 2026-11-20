// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test16

// Level is parsed from text.
type Level int32

func (l *Level) UnmarshalText(_ []byte) error { return nil }

type Mode uint8

func (m Mode) UnmarshalText(_ []byte) error { return nil }

type Plain int32

type TextUnmarshalerConfig struct {
	Level Level `mapstructure:"level"`
	Mode  Mode  `mapstructure:"mode"`
	Plain Plain `mapstructure:"plain"`
}
