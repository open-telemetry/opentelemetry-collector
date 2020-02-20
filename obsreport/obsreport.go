// Copyright 2020 OpenTelemetry Authors
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

package obsreport

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/observability"
)

const (
	nameSep = "/"
)

var (
	// Variables to control the usage of legacy and new metrics.
	useLegacy = true
	useNew    = true

	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

// Configure is used to control the settings that will be used by the obsreport
// package.
func Configure(
	generateLegacy, generateNew bool,
) (views []*view.View) {

	// TODO: expose some level control, similar to telemetry.Level

	useLegacy = generateLegacy
	useNew = generateNew

	if useLegacy {
		views = append(views, observability.AllViews...)
	}

	if useNew {
		views = append(views, genAllViews()...)
	}

	return views
}

func genAllViews() (views []*view.View) {
	// Receiver views.
	measures := []*stats.Int64Measure{
		mReceiverAcceptedSpans, mReceiverRefusedSpans,
		mReceiverAcceptedMetricPoints, mReceiverRefusedMetricPoints,
	}
	tagKeys := []tag.Key{
		tagKeyReceiver, tagKeyTransport,
	}
	views = append(views, genViews(
		measures, tagKeys, view.Sum())...)

	// Exporter views.
	measures = []*stats.Int64Measure{
		mExporterSentSpans, mExporterFailedToSendSpans,
		mExporterSentMetricPoints, mExporterFailedToSendMetricPoints,
	}
	tagKeys = []tag.Key{tagKeyExporter}
	views = append(views, genViews(
		measures, tagKeys, view.Sum())...)

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

func errToStatus(err error) trace.Status {
	if err != nil {
		return trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()}
	}
	return okStatus
}
