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
	*pMetric
}

type pMetric struct {
	orig  *otlpcollectormetrics.ExportMetricsServiceRequest
	state State
}

func NewMetricsFromResourceMetricsOrig(orig *[]*otlpmetrics.ResourceMetrics) Metrics {
	return Metrics{&pMetric{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{ResourceMetrics: *orig}, state: StateExclusive}}
}

func (ms Metrics) IsShared() bool {
	return ms.pMetric != nil && ms.state == StateShared
}

func (ms Metrics) MarkExclusive() {
	ms.state = StateExclusive
}

func (ms Metrics) AsShared() Metrics {
	ms.state = StateShared
	return Metrics{&pMetric{orig: ms.orig, state: StateShared}}
}

func (ms Metrics) GetOrig() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return ms.orig
}

func (ms Metrics) SetOrig(orig *otlpcollectormetrics.ExportMetricsServiceRequest) {
	ms.orig = orig
}

func NewMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest) Metrics {
	return Metrics{&pMetric{orig: orig, state: StateExclusive}}
}

// MetricsToProto internal helper to convert Metrics to protobuf representation.
func MetricsToProto(l Metrics) otlpmetrics.MetricsData {
	return otlpmetrics.MetricsData{
		ResourceMetrics: l.orig.ResourceMetrics,
	}
}

// MetricsFromProto internal helper to convert protobuf representation to Metrics.
func MetricsFromProto(orig otlpmetrics.MetricsData) Metrics {
	return Metrics{&pMetric{
		orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
			ResourceMetrics: orig.ResourceMetrics,
		},
		state: StateExclusive,
	}}
}
