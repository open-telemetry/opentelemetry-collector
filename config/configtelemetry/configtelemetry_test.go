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
	"github.com/stretchr/testify/require"
)

func TestParseFrom(t *testing.T) {
	tests := []struct {
		str   string
		level Level
		err   bool
	}{
		{
			str:   "",
			level: LevelNone,
			err:   true,
		},
		{
			str:   "other_string",
			level: LevelNone,
			err:   true,
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
			lvl, err := parseLevel(test.str)
			if test.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.level, lvl)
		})
	}
}

func TestLevelSet(t *testing.T) {
	tests := []struct {
		str   string
		level Level
		err   bool
	}{
		{
			str:   "",
			level: LevelNone,
			err:   true,
		},
		{
			str:   "other_string",
			level: LevelNone,
			err:   true,
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
			lvl := new(Level)
			err := lvl.Set(test.str)
			if test.err {
				assert.Error(t, err)
				assert.Equal(t, LevelBasic, *lvl)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.level, *lvl)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		str   string
		level Level
		err   bool
	}{
		{
			str:   "unknown",
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
		})
	}
}

func TestTelemetrySettings(t *testing.T) {
	ts := &TelemetrySetting{
		MetricsLevelStr: "unknown",
	}
	_, err := ts.GetMetricsLevel()
	assert.Error(t, err)
}

func TestDefaultTelemetrySettings(t *testing.T) {
	ts := DefaultTelemetrySetting()
	assert.Equal(t, levelBasicStr, ts.MetricsLevelStr)
	lvl, err := ts.GetMetricsLevel()
	require.NoError(t, err)
	assert.Equal(t, LevelBasic, lvl)
}
