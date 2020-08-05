// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"testing"

	"github.com/bmizerany/assert"
	"github.com/go-kit/kit/log/level"
	"go.uber.org/zap/zapcore"
)

func TestExtractLogLevel(t *testing.T) {
	tcs := []struct {
		name       string
		input      []interface{}
		wantLevel  zapcore.Level
		wantOutput []interface{}
	}{
		{
			name:       "nil fields",
			input:      nil,
			wantLevel:  zapcore.InfoLevel, // Default
			wantOutput: []interface{}{},
		},
		{
			name:       "empty fields",
			input:      []interface{}{},
			wantLevel:  zapcore.InfoLevel, // Default
			wantOutput: []interface{}{},
		},
		{
			name: "warn level",
			input: []interface{}{
				"level",
				level.WarnValue(),
			},
			wantLevel:  zapcore.WarnLevel,
			wantOutput: []interface{}{},
		},
		{
			name: "debug level + extra fields",
			input: []interface{}{
				"ts",
				1596604719.955769,
				"level",
				level.DebugValue(),
				"msg",
				"http client error",
			},
			wantLevel: zapcore.DebugLevel,
			wantOutput: []interface{}{
				"ts",
				1596604719.955769,
				"msg",
				"http client error",
			},
		},
		{
			name: "missing level field",
			input: []interface{}{
				"ts",
				1596604719.955769,
				"msg",
				"http client error",
			},
			wantLevel: zapcore.InfoLevel, // Default
			wantOutput: []interface{}{
				"ts",
				1596604719.955769,
				"msg",
				"http client error",
			},
		},
		{
			name: "invalid level type",
			input: []interface{}{
				"level",
				"warn", // String is not recognized
			},
			wantLevel: zapcore.InfoLevel, // Default
			wantOutput: []interface{}{
				"level",
				"warn", // Field is preserved
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			zapLevel, output := extractLogLevel(tc.input)
			assert.Equal(t, tc.wantLevel, zapLevel)
			assert.Equal(t, tc.wantOutput, output)
		})
	}
}
