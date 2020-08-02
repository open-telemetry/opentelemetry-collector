// Copyright 2020 The OpenTelemetry Authors
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

package cortexexporter

import (
	"github.com/prometheus/prometheus/prompb"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
)

// check whether the metric has the correct type and kind combination
func validateMetrics(desc *otlp.MetricDescriptor) bool {
	switch desc.GetType() {
		case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_MONOTONIC_INT64,
		otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_SUMMARY:
			return desc.GetTemporality() == otlp.MetricDescriptor_CUMULATIVE
		case otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_DOUBLE:
			return true
	}
	return false
}

func addSample(tsMap map[string]*prompb.TimeSeries, sample *prompb.Sample, lbs []prompb.Label,desc otlp.MetricDescriptor_Type){}

func timeSeriesSignature(t otlp.MetricDescriptor_Type, lbs *[]prompb.Label) string {return ""}

// sanitize labels as well; label in extra ovewrites label in labels if collision happens, perhaps log the overwrite
func createLabelSet(labels []*common.StringKeyValue, extras ...string) []prompb.Label { return nil}


func handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) (error) {return nil}