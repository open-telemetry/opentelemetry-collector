// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
