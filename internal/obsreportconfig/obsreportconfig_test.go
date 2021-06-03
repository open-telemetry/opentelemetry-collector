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
	"go.opencensus.io/stats/view"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name      string
		level     configtelemetry.Level
		wantViews []*view.View
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:      "basic",
			level:     configtelemetry.LevelBasic,
			wantViews: allViews().Views,
		},
		{
			name:      "normal",
			level:     configtelemetry.LevelNormal,
			wantViews: allViews().Views,
		},
		{
			name:      "detailed",
			level:     configtelemetry.LevelDetailed,
			wantViews: allViews().Views,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotViews := Configure(tt.level)
			assert.Equal(t, tt.wantViews, gotViews.Views)
		})
	}
}
