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

package obsreportconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name         string
		level        configtelemetry.Level
		wantViewsLen int
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:         "basic",
			level:        configtelemetry.LevelBasic,
			wantViewsLen: 24,
		},
		{
			name:         "normal",
			level:        configtelemetry.LevelNormal,
			wantViewsLen: 24,
		},
		{
			name:         "detailed",
			level:        configtelemetry.LevelDetailed,
			wantViewsLen: 24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, AllViews(tt.level), tt.wantViewsLen)
		})
	}
}
