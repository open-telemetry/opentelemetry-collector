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

package internal

import (
	"testing"

	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLog(t *testing.T) {
	tcs := []struct {
		name        string
		input       []interface{}
		wantLevel   zapcore.Level
		wantMessage string
	}{
		{
			name: "Starting provider",
			input: []interface{}{
				"level",
				level.DebugValue(),
				"msg",
				"Starting provider",
				"provider",
				"string/0",
				"subs",
				"[target1]",
			},
			wantLevel:   zapcore.DebugLevel,
			wantMessage: "Starting provider",
		},
		{
			name: "Scrape failed",
			input: []interface{}{
				"level",
				level.ErrorValue(),
				"scrape_pool",
				"target1",
				"msg",
				"Scrape failed",
				"err",
				"server returned HTTP status 500 Internal Server Error",
			},
			wantLevel:   zapcore.ErrorLevel,
			wantMessage: "Scrape failed",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			conf := zap.NewProductionConfig()
			conf.Level.SetLevel(zapcore.DebugLevel)

			// capture zap log entry
			var entry zapcore.Entry
			h := func(e zapcore.Entry) error {
				entry = e
				return nil
			}

			logger, err := conf.Build(zap.Hooks(h))
			require.NoError(t, err)

			adapter := NewZapToGokitLogAdapter(logger)
			err = adapter.Log(tc.input...)
			require.NoError(t, err)

			assert.Equal(t, tc.wantLevel, entry.Level)
			assert.Equal(t, tc.wantMessage, entry.Message)
		})
	}
}

func TestExtractLogData(t *testing.T) {
	tcs := []struct {
		name        string
		input       []interface{}
		wantLevel   level.Value
		wantMessage string
		wantOutput  []interface{}
	}{
		{
			name:        "nil fields",
			input:       nil,
			wantLevel:   level.InfoValue(), // Default
			wantMessage: "",
			wantOutput:  []interface{}{},
		},
		{
			name:        "empty fields",
			input:       []interface{}{},
			wantLevel:   level.InfoValue(), // Default
			wantMessage: "",
			wantOutput:  []interface{}{},
		},
		{
			name: "info level",
			input: []interface{}{
				"level",
				level.InfoValue(),
			},
			wantLevel:   level.InfoValue(),
			wantMessage: "",
			wantOutput:  []interface{}{},
		},
		{
			name: "warn level",
			input: []interface{}{
				"level",
				level.WarnValue(),
			},
			wantLevel:   level.WarnValue(),
			wantMessage: "",
			wantOutput:  []interface{}{},
		},
		{
			name: "error level",
			input: []interface{}{
				"level",
				level.ErrorValue(),
			},
			wantLevel:   level.ErrorValue(),
			wantMessage: "",
			wantOutput:  []interface{}{},
		},
		{
			name: "debug level + extra fields",
			input: []interface{}{
				"timestamp",
				1596604719,
				"level",
				level.DebugValue(),
				"msg",
				"http client error",
			},
			wantLevel:   level.DebugValue(),
			wantMessage: "http client error",
			wantOutput: []interface{}{
				"timestamp",
				1596604719,
			},
		},
		{
			name: "missing level field",
			input: []interface{}{
				"timestamp",
				1596604719,
				"msg",
				"http client error",
			},
			wantLevel:   level.InfoValue(), // Default
			wantMessage: "http client error",
			wantOutput: []interface{}{
				"timestamp",
				1596604719,
			},
		},
		{
			name: "invalid level type",
			input: []interface{}{
				"level",
				"warn", // String is not recognized
			},
			wantLevel: level.InfoValue(), // Default
			wantOutput: []interface{}{
				"level",
				"warn", // Field is preserved
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ld := extractLogData(tc.input)
			assert.Equal(t, tc.wantLevel, ld.level)
			assert.Equal(t, tc.wantMessage, ld.msg)
			assert.Equal(t, tc.wantOutput, ld.otherFields)
		})
	}
}
