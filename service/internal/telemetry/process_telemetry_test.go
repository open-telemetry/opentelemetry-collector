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

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
)

func TestProcessTelemetry(t *testing.T) {
	const ballastSizeBytes uint64 = 0
	pmv, err := NewProcessMetricsViews(ballastSizeBytes)
	require.NoError(t, err)
	assert.NotNil(t, pmv)

	expectedViews := []string{
		// Changing a metric name is a breaking change.
		// Adding new metrics is ok as long it follows the conventions described at
		// https://pkg.go.dev/go.opentelemetry.io/collector/obsreport?tab=doc#hdr-Naming_Convention_for_New_Metrics
		"process/uptime",
		"process/runtime/heap_alloc_bytes",
		"process/runtime/total_alloc_bytes",
		"process/runtime/total_sys_memory_bytes",
		"process/cpu_seconds",
		"process/memory/rss",
	}
	processViews := pmv.Views()
	assert.Len(t, processViews, len(expectedViews))

	require.NoError(t, view.Register(processViews...))
	defer view.Unregister(processViews...)

	// Check that the views are actually filled.
	pmv.updateViews()
	<-time.After(200 * time.Millisecond)

	for _, viewName := range expectedViews {
		rows, err := view.RetrieveData(viewName)
		require.NoError(t, err, viewName)

		require.Len(t, rows, 1, viewName)
		row := rows[0]
		assert.Len(t, row.Tags, 0)

		var value float64
		if viewName == "process/uptime" {
			value = row.Data.(*view.SumData).Value
		} else {
			value = row.Data.(*view.LastValueData).Value
		}

		if viewName == "process/uptime" || viewName == "process/cpu_seconds" {
			// This likely will still be zero when running the test.
			assert.True(t, value >= 0, viewName)
			continue
		}

		assert.True(t, value > 0, viewName)
	}
}
