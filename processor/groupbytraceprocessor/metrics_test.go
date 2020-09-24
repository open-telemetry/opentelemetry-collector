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

package groupbytraceprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestProcessorMetrics(t *testing.T) {
	tests := []struct {
		viewNames []string
		level     configtelemetry.Level
	}{
		{
			viewNames: []string{
				"conf_num_traces",
				"num_events_in_queue",
				"num_traces_in_memory",
				"traces_evicted",
				"spans_released",
				"traces_released",
				"incomplete_releases",
				"event_latency",
			},
			level: configtelemetry.LevelDetailed,
		},
		{
			viewNames: []string{},
			level:     configtelemetry.LevelNone,
		},
	}
	for _, test := range tests {
		views := MetricViews(test.level)
		if test.viewNames == nil {
			assert.Nil(t, views)
			continue
		}
		for i, viewName := range test.viewNames {
			assert.Equal(t, viewName, views[i].Name)
		}
	}
}
