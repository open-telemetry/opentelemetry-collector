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
	*pMetrics
}

type pMetrics struct {
	orig  *otlpcollectormetrics.ExportMetricsServiceRequest
	state *State
}

func (ms Metrics) AsShared() Metrics {
	*ms.state = StateShared
	state := StateShared
	return Metrics{&pMetrics{orig: ms.orig, state: &state}}
}

func (ms Metrics) GetState() *State {
	return ms.state
}

func (ms Metrics) SetState(state *State) {
	ms.state = state
}

func (ms Metrics) GetOrig() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return ms.orig
}

func (ms Metrics) SetOrig(orig *otlpcollectormetrics.ExportMetricsServiceRequest) {
	ms.pMetrics.orig = orig
}

func NewMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest) Metrics {
	state := StateExclusive
	return Metrics{&pMetrics{orig: orig, state: &state}}
}

func NewMetricsFromResourceMetricsOrig(orig *[]*otlpmetrics.ResourceMetrics) Metrics {
	state := StateExclusive
	return Metrics{&pMetrics{
		orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
			ResourceMetrics: *orig,
		},
		state: &state,
	}}
}

// MetricsToProto internal helper to convert Metrics to protobuf representation.
func MetricsToProto(l Metrics) otlpmetrics.MetricsData {
	return otlpmetrics.MetricsData{
		ResourceMetrics: l.orig.ResourceMetrics,
	}
}

// MetricsFromProto internal helper to convert protobuf representation to Metrics.
func MetricsFromProto(orig otlpmetrics.MetricsData) Metrics {
	state := StateExclusive
	return Metrics{&pMetrics{
		orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
			ResourceMetrics: orig.ResourceMetrics,
		},
		state: &state,
	}}
}
