// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

type Metrics struct {
	orig  *otlpcollectormetrics.ExportMetricsServiceRequest
	state State
}

func GetOrigMetrics(ms Metrics) *otlpcollectormetrics.ExportMetricsServiceRequest {
	return ms.orig
}

func GetMetricsState(ms Metrics) State {
	return ms.state
}

func SetMetricsState(ms Metrics, s State) {
	ms.state = s
}

func NewMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest, s State) Metrics {
	return Metrics{orig: orig, state: s}
}

// MetricsToProto internal helper to convert Metrics to protobuf representation.
func MetricsToProto(l Metrics) otlpmetrics.MetricsData {
	return otlpmetrics.MetricsData{
		ResourceMetrics: l.orig.ResourceMetrics,
	}
}

// MetricsFromProto internal helper to convert protobuf representation to Metrics.
func MetricsFromProto(orig otlpmetrics.MetricsData) Metrics {
	return Metrics{
		orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
			ResourceMetrics: orig.ResourceMetrics,
		},
		state: StateShared,
	}
}
