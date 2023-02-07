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
	sm *stateMetrics
}

type stateMetrics struct {
	orig  *otlpcollectormetrics.ExportMetricsServiceRequest
	state State
}

func GetMetricsOrig(ms Metrics) *otlpcollectormetrics.ExportMetricsServiceRequest {
	return ms.sm.orig
}

func GetMetricsState(ms Metrics) State {
	return ms.sm.state
}

// ResetStateMetrics replaces the internal StateMetrics with a new empty and the provided state.
func ResetStateMetrics(ms Metrics, s State) {
	ms.sm.orig = &otlpcollectormetrics.ExportMetricsServiceRequest{}
	ms.sm.state = s
}

func NewMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest, s State) Metrics {
	return Metrics{&stateMetrics{orig: orig, state: s}}
}

// MetricsToProto internal helper to convert Metrics to protobuf representation.
func MetricsToProto(ms Metrics) otlpmetrics.MetricsData {
	return otlpmetrics.MetricsData{
		ResourceMetrics: ms.sm.orig.ResourceMetrics,
	}
}

// MetricsFromProto internal helper to convert protobuf representation to Metrics.
func MetricsFromProto(orig otlpmetrics.MetricsData) Metrics {
	return Metrics{&stateMetrics{
		orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
			ResourceMetrics: orig.ResourceMetrics,
		},
		state: StateExclusive,
	}}
}
