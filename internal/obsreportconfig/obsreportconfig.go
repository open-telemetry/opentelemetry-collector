// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreportconfig // import "go.opentelemetry.io/collector/internal/obsreportconfig"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// UseOtelForInternalMetricsfeatureGate is the feature gate that controls whether the collector uses open
// telemetrySettings for internal metrics.
var UseOtelForInternalMetricsfeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.useOtelForInternalMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector uses OpenTelemetry for internal metrics"))

// DisableHighCardinalityMetricsfeatureGate is the feature gate that controls whether the collector should enable
// potentially high cardinality metrics. The gate will be removed when the collector allows for view configuration.
var DisableHighCardinalityMetricsfeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableHighCardinalityMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector should enable potentially high"+
		"cardinality metrics. The gate will be removed when the collector allows for view configuration."))

// UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate is the feature gate that controls whether the collector
// supports configuring the OpenTelemetry SDK via configuration
var UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.useOtelWithSDKConfigurationForInternalTelemetry",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector supports extended OpenTelemetry"+
		"configuration for internal telemetry"))

// AllViews returns all the OpenCensus views requires by obsreport package.
func AllViews(level configtelemetry.Level) []*view.View {
	if level == configtelemetry.LevelNone {
		return nil
	}

	var views []*view.View
	var measures []*stats.Int64Measure
	var tagKeys []tag.Key

	// Receiver views.
	views = append(views, receiverViews()...)

	// Scraper views.
	views = append(views, scraperViews()...)

	// Exporter views.
	measures = []*stats.Int64Measure{
		obsmetrics.ExporterSentSpans,
		obsmetrics.ExporterFailedToSendSpans,
		obsmetrics.ExporterSentMetricPoints,
		obsmetrics.ExporterFailedToSendMetricPoints,
		obsmetrics.ExporterSentLogRecords,
		obsmetrics.ExporterFailedToSendLogRecords,
	}
	tagKeys = []tag.Key{obsmetrics.TagKeyExporter}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)

	errorNumberView := &view.View{
		Name:        obsmetrics.ExporterPrefix + "send_failed_requests",
		Description: "number of times exporters failed to send requests to the destination",
		Measure:     obsmetrics.ExporterFailedToSendSpans,
		Aggregation: view.Count(),
	}
	views = append(views, errorNumberView)

	// Processor views.
	measures = []*stats.Int64Measure{
		obsmetrics.ProcessorAcceptedSpans,
		obsmetrics.ProcessorRefusedSpans,
		obsmetrics.ProcessorDroppedSpans,
		obsmetrics.ProcessorAcceptedMetricPoints,
		obsmetrics.ProcessorRefusedMetricPoints,
		obsmetrics.ProcessorDroppedMetricPoints,
		obsmetrics.ProcessorAcceptedLogRecords,
		obsmetrics.ProcessorRefusedLogRecords,
		obsmetrics.ProcessorDroppedLogRecords,
	}
	tagKeys = []tag.Key{obsmetrics.TagKeyProcessor}
	views = append(views, genViews(measures, tagKeys, view.Sum())...)

	return views
}

func receiverViews() []*view.View {
	if UseOtelForInternalMetricsfeatureGate.IsEnabled() {
		return nil
	}

	measures := []*stats.Int64Measure{
		obsmetrics.ReceiverAcceptedSpans,
		obsmetrics.ReceiverRefusedSpans,
		obsmetrics.ReceiverAcceptedMetricPoints,
		obsmetrics.ReceiverRefusedMetricPoints,
		obsmetrics.ReceiverAcceptedLogRecords,
		obsmetrics.ReceiverRefusedLogRecords,
	}
	tagKeys := []tag.Key{
		obsmetrics.TagKeyReceiver, obsmetrics.TagKeyTransport,
	}

	return genViews(measures, tagKeys, view.Sum())
}

func scraperViews() []*view.View {
	if UseOtelForInternalMetricsfeatureGate.IsEnabled() {
		return nil
	}

	measures := []*stats.Int64Measure{
		obsmetrics.ScraperScrapedMetricPoints,
		obsmetrics.ScraperErroredMetricPoints,
	}
	tagKeys := []tag.Key{obsmetrics.TagKeyReceiver, obsmetrics.TagKeyScraper}

	return genViews(measures, tagKeys, view.Sum())
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
