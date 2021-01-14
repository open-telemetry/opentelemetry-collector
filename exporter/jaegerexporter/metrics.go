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

package jaegerexporter

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	mLastConnectionState = stats.Int64("jaegerexporter_conn_state", "Last connection state: 0 = Idle, 1 = Connecting, 2 = Ready, 3 = TransientFailure, 4 = Shutdown", stats.UnitDimensionless)
	vLastConnectionState = &view.View{
		Name:        mLastConnectionState.Name(),
		Measure:     mLastConnectionState,
		Description: mLastConnectionState.Description(),
		Aggregation: view.LastValue(),
		TagKeys: []tag.Key{
			tag.MustNewKey("exporter_name"),
		},
	}
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews() []*view.View {
	return []*view.View{vLastConnectionState}
}
