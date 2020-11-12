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

func TestParseLevel(t *testing.T) {
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
			str:   "none",
			level: LevelNone,
		},
		{
			str:   "basic",
			level: LevelBasic,
		},
		{
			str:   "normal",
			level: LevelNormal,
		},
		{
			str:   "detailed",
			level: LevelDetailed,
		},
	}

	for _, test := range tests {
		t.Run(test.str, func(t *testing.T) {
			lvl, err := ParseLevel(test.str)
			if test.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.level, lvl)
		})
	}
}
