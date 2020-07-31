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

package obsreport

import (
	"strings"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/consumer/consumerdata"
)

const (
	nameSep = "/"
)

var (
	// Variables to control the usage of legacy and new metrics.
	useLegacy = true
	useNew    = true
)

// Configure is used to control the settings that will be used by the obsreport
// package.
func Configure(generateLegacy, generateNew bool) (views []*view.View) {

	// TODO: expose some level control, similar to telemetry.Level

	useLegacy = generateLegacy
	useNew = generateNew

	if useLegacy {
		views = append(views, LegacyAllViews...)
	}

	if useNew {
		views = append(views, AllViews()...)
	}

	return views
}

// CountMetricPoints is a helper to count the "amount" of metrics data. For code using the
// internal data structure, pdata.Metrics.MetricAndDataPointCount should be used instead
func CountMetricPoints(md consumerdata.MetricsData) (numTimeSeries int, numPoints int) {
	for _, metric := range md.Metrics {
		tss := metric.GetTimeseries()
		numTimeSeries += len(tss)
		for _, ts := range tss {
			numPoints += len(ts.GetPoints())
		}
	}
	return numTimeSeries, numPoints
}

func buildComponentPrefix(componentPrefix, configType string) string {
	if !strings.HasSuffix(componentPrefix, nameSep) {
		componentPrefix += nameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + nameSep
}

// AllViews return the list of all views that needs to be configured.
func AllViews() (views []*view.View) {
	// Receiver views.
	measures := []*stats.Int64Measure{
		mReceiverAcceptedSpans,
		mReceiverRefusedSpans,
		mReceiverAcceptedMetricPoints,
		mReceiverRefusedMetricPoints,
		mReceiverAcceptedLogRecords,
		mReceiverRefusedLogRecords,
	}
	tagKeys := []tag.Key{
		tagKeyReceiver, tagKeyTransport,
	}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)

	// Exporter views.
	measures = []*stats.Int64Measure{
		mExporterSentSpans,
		mExporterFailedToSendSpans,
		mExporterSentMetricPoints,
		mExporterFailedToSendMetricPoints,
		mExporterSentLogRecords,
		mExporterFailedToSendLogRecords,
	}
	tagKeys = []tag.Key{tagKeyExporter}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)

	// Processor views.
	measures = []*stats.Int64Measure{
		mProcessorAcceptedSpans,
		mProcessorRefusedSpans,
		mProcessorDroppedSpans,
		mProcessorAcceptedMetricPoints,
		mProcessorRefusedMetricPoints,
		mProcessorDroppedMetricPoints,
		mProcessorAcceptedLogRecords,
		mProcessorRefusedLogRecords,
		mProcessorDroppedLogRecords,
	}
	tagKeys = []tag.Key{tagKeyProcessor}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)

	return views
}

func genViews(
	measures []*stats.Int64Measure,
	tagKeys []tag.Key,
	aggregation *view.Aggregation,
) []*view.View {
	views := make([]*view.View, 0, len(measures))
	for _, measure := range measures {
		views = append(views, &view.View{
			Name:        measure.Name(),
			Description: measure.Description(),
			TagKeys:     tagKeys,
			Measure:     measure,
			Aggregation: aggregation,
		})
	}
	return views
}
