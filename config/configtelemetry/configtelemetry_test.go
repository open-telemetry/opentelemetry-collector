// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtelemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalText(t *testing.T) {
	tests := []struct {
		str   []string
		level Level
		err   bool
	}{
		{
			str:   []string{"", "other_string"},
			level: LevelNone,
			err:   true,
		},
		{
			str:   []string{"none", "None", "NONE"},
			level: LevelNone,
		},
		{
			str:   []string{"basic", "Basic", "BASIC"},
			level: LevelBasic,
		},
		{
			str:   []string{"normal", "Normal", "NORMAL"},
			level: LevelNormal,
		},
		{
			str:   []string{"detailed", "Detailed", "DETAILED"},
			level: LevelDetailed,
		},
	}

	for _, test := range tests {
		for _, str := range test.str {
			t.Run(str, func(t *testing.T) {
				var lvl Level
				err := lvl.UnmarshalText([]byte(str))
				if test.err {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, test.level, lvl)
				}
			})
		}
	}
}

func TestUnmarshalTextNilLevel(t *testing.T) {
	lvl := (*Level)(nil)
	assert.Error(t, lvl.UnmarshalText([]byte(levelNormalStr)))
}

func TestLevelStringMarshal(t *testing.T) {
	tests := []struct {
		str   string
		level Level
		err   bool
	}{
		{
			str:   "",
			level: Level(-10),
		},
		{
			str:   levelNoneStr,
			level: LevelNone,
		},
		{
			str:   levelBasicStr,
			level: LevelBasic,
		},
		{
			str:   levelNormalStr,
			level: LevelNormal,
		},
		{
			str:   levelDetailedStr,
			level: LevelDetailed,
		},
	}
	for _, test := range tests {
		t.Run(test.str, func(t *testing.T) {
			assert.Equal(t, test.str, test.level.String())
			got, err := test.level.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, test.str, string(got))
		})
	}
}
