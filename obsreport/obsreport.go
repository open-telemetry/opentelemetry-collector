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
	"context"
	"strings"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

const (
	nameSep = "/"
)

var (
	gLevel = configtelemetry.LevelBasic

	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

// setParentLink tries to retrieve a span from parentCtx and if one exists
// sets its SpanID, TraceID as a link to the given child Span.
// It returns true only if it retrieved a parent span from the context.
//
// This is typically used when the parentCtx may already have a trace and is
// long lived (eg.: an gRPC stream, or TCP connection) and one desires distinct
// traces for individual operations under the long lived trace associated to
// the parentCtx. This function is a helper that encapsulates the work of
// linking the short lived trace/span to the longer one.
func setParentLink(parentCtx context.Context, childSpan *trace.Span) bool {
	parentSpanFromRPC := trace.FromContext(parentCtx)
	if parentSpanFromRPC == nil {
		return false
	}

	psc := parentSpanFromRPC.SpanContext()
	childSpan.AddLink(trace.Link{
		SpanID:  psc.SpanID,
		TraceID: psc.TraceID,
		Type:    trace.LinkTypeParent,
	})
	return true
}

// Configure is used to control the settings that will be used by the obsreport
// package.
func Configure(level configtelemetry.Level) (views []*view.View) {
	gLevel = level

	if gLevel != configtelemetry.LevelNone {
		gProcessorObsReport.level = level
		views = append(views, AllViews()...)
	}

	return views
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

	// Scraper views.
	measures = []*stats.Int64Measure{
		mScraperScrapedMetricPoints,
		mScraperErroredMetricPoints,
	}
	tagKeys = []tag.Key{tagKeyReceiver, tagKeyScraper}
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

func errToStatus(err error) trace.Status {
	if err != nil {
		return trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()}
	}
	return okStatus
}
