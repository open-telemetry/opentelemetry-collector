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

package obsreporttest // import "go.opentelemetry.io/collector/obsreport/obsreporttest"

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

var (
	// Names used by the metrics and labels are hard coded here in order to avoid
	// inadvertent changes: at this point changing metric names and labels should
	// be treated as a breaking changing and requires a good justification.
	// Changes to metric names or labels can break alerting, dashboards, etc
	// that are used to monitor the Collector in production deployments.
	// DO NOT SWITCH THE VARIABLES BELOW TO SIMILAR ONES DEFINED ON THE PACKAGE.
	receiverTag, _  = tag.NewKey("receiver")
	scraperTag, _   = tag.NewKey("scraper")
	transportTag, _ = tag.NewKey("transport")
	exporterTag, _  = tag.NewKey("exporter")
	processorTag, _ = tag.NewKey("processor")
)

type TestTelemetry struct {
	component.TelemetrySettings
	SpanRecorder *tracetest.SpanRecorder
	views        []*view.View
}

// ToExporterCreateSettings returns ExporterCreateSettings with configured TelemetrySettings
func (tts *TestTelemetry) ToExporterCreateSettings() component.ExporterCreateSettings {
	exporterSettings := componenttest.NewNopExporterCreateSettings()
	exporterSettings.TelemetrySettings = tts.TelemetrySettings
	return exporterSettings
}

// ToProcessorCreateSettings returns ProcessorCreateSettings with configured TelemetrySettings
func (tts *TestTelemetry) ToProcessorCreateSettings() component.ProcessorCreateSettings {
	processorSettings := componenttest.NewNopProcessorCreateSettings()
	processorSettings.TelemetrySettings = tts.TelemetrySettings
	return processorSettings
}

// ToReceiverCreateSettings returns ReceiverCreateSettings with configured TelemetrySettings
func (tts *TestTelemetry) ToReceiverCreateSettings() component.ReceiverCreateSettings {
	receiverSettings := componenttest.NewNopReceiverCreateSettings()
	receiverSettings.TelemetrySettings = tts.TelemetrySettings
	return receiverSettings
}

// Shutdown unregisters any views and shuts down the SpanRecorder
func (tts *TestTelemetry) Shutdown(ctx context.Context) error {
	view.Unregister(tts.views...)
	return tts.SpanRecorder.Shutdown(ctx)
}

// SetupTelemetry does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The caller should defer a call to Shutdown the returned TestTelemetry.
func SetupTelemetry() (TestTelemetry, error) {
	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	settings := TestTelemetry{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		SpanRecorder:      sr,
	}
	settings.TracerProvider = tp
	obsMetrics := obsreportconfig.Configure(configtelemetry.LevelNormal)
	settings.views = obsMetrics.Views
	err := view.Register(settings.views...)
	if err != nil {
		return settings, err
	}

	return settings, err
}

// CheckExporterTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckExporterTraces(_ TestTelemetry, exporter config.ComponentID, sentSpans, sendFailedSpans int64) error {
	exporterTags := tagsForExporterView(exporter)
	if sendFailedSpans > 0 {
		return multierr.Combine(
			checkValueForView(exporterTags, sentSpans, "exporter/sent_spans"),
			checkValueForView(exporterTags, sendFailedSpans, "exporter/send_failed_spans"))
	}
	return checkValueForView(exporterTags, sentSpans, "exporter/sent_spans")
}

// CheckExporterMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckExporterMetrics(_ TestTelemetry, exporter config.ComponentID, sentMetricsPoints, sendFailedMetricsPoints int64) error {
	exporterTags := tagsForExporterView(exporter)
	if sendFailedMetricsPoints > 0 {
		return multierr.Combine(
			checkValueForView(exporterTags, sentMetricsPoints, "exporter/sent_metric_points"),
			checkValueForView(exporterTags, sendFailedMetricsPoints, "exporter/send_failed_metric_points"))
	}
	return checkValueForView(exporterTags, sentMetricsPoints, "exporter/sent_metric_points")
}

// CheckExporterLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckExporterLogs(_ TestTelemetry, exporter config.ComponentID, sentLogRecords, sendFailedLogRecords int64) error {
	exporterTags := tagsForExporterView(exporter)
	if sendFailedLogRecords > 0 {
		return multierr.Combine(
			checkValueForView(exporterTags, sentLogRecords, "exporter/sent_log_records"),
			checkValueForView(exporterTags, sendFailedLogRecords, "exporter/send_failed_log_records"))
	}
	return checkValueForView(exporterTags, sentLogRecords, "exporter/sent_log_records")
}

// CheckProcessorTraces checks that for the current exported values for trace exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckProcessorTraces(_ TestTelemetry, processor config.ComponentID, acceptedSpans, refusedSpans, droppedSpans int64) error {
	processorTags := tagsForProcessorView(processor)
	return multierr.Combine(
		checkValueForView(processorTags, acceptedSpans, "processor/accepted_spans"),
		checkValueForView(processorTags, refusedSpans, "processor/refused_spans"),
		checkValueForView(processorTags, droppedSpans, "processor/dropped_spans"))
}

// CheckProcessorMetrics checks that for the current exported values for metrics exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckProcessorMetrics(_ TestTelemetry, processor config.ComponentID, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints int64) error {
	processorTags := tagsForProcessorView(processor)
	return multierr.Combine(
		checkValueForView(processorTags, acceptedMetricPoints, "processor/accepted_metric_points"),
		checkValueForView(processorTags, refusedMetricPoints, "processor/refused_metric_points"),
		checkValueForView(processorTags, droppedMetricPoints, "processor/dropped_metric_points"))
}

// CheckProcessorLogs checks that for the current exported values for logs exporter metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckProcessorLogs(_ TestTelemetry, processor config.ComponentID, acceptedLogRecords, refusedLogRecords, droppedLogRecords int64) error {
	processorTags := tagsForProcessorView(processor)
	return multierr.Combine(
		checkValueForView(processorTags, acceptedLogRecords, "processor/accepted_log_records"),
		checkValueForView(processorTags, refusedLogRecords, "processor/refused_log_records"),
		checkValueForView(processorTags, droppedLogRecords, "processor/dropped_log_records"))
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckReceiverTraces(_ TestTelemetry, receiver config.ComponentID, protocol string, acceptedSpans, droppedSpans int64) error {
	receiverTags := tagsForReceiverView(receiver, protocol)
	return multierr.Combine(
		checkValueForView(receiverTags, acceptedSpans, "receiver/accepted_spans"),
		checkValueForView(receiverTags, droppedSpans, "receiver/refused_spans"))
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckReceiverLogs(_ TestTelemetry, receiver config.ComponentID, protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	receiverTags := tagsForReceiverView(receiver, protocol)
	return multierr.Combine(
		checkValueForView(receiverTags, acceptedLogRecords, "receiver/accepted_log_records"),
		checkValueForView(receiverTags, droppedLogRecords, "receiver/refused_log_records"))
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckReceiverMetrics(_ TestTelemetry, receiver config.ComponentID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverTags := tagsForReceiverView(receiver, protocol)
	return multierr.Combine(
		checkValueForView(receiverTags, acceptedMetricPoints, "receiver/accepted_metric_points"),
		checkValueForView(receiverTags, droppedMetricPoints, "receiver/refused_metric_points"))
}

// CheckScraperMetrics checks that for the current exported values for metrics scraper metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func CheckScraperMetrics(_ TestTelemetry, receiver config.ComponentID, scraper config.ComponentID, scrapedMetricPoints, erroredMetricPoints int64) error {
	scraperTags := tagsForScraperView(receiver, scraper)
	return multierr.Combine(
		checkValueForView(scraperTags, scrapedMetricPoints, "scraper/scraped_metric_points"),
		checkValueForView(scraperTags, erroredMetricPoints, "scraper/errored_metric_points"))
}

// checkValueForView checks that for the current exported value in the view with the given name
// for {LegacyTagKeyReceiver: receiverName} is equal to "value".
func checkValueForView(wantTags []tag.Tag, value int64, vName string) error {
	// Make sure the tags slice is sorted by tag keys.
	sortTags(wantTags)

	rows, err := view.RetrieveData(vName)
	if err != nil {
		return err
	}

	for _, row := range rows {
		// Make sure the tags slice is sorted by tag keys.
		sortTags(row.Tags)
		if reflect.DeepEqual(wantTags, row.Tags) {
			sum := row.Data.(*view.SumData)
			if float64(value) != sum.Value {
				return fmt.Errorf("[%s]: values did no match, wanted %f got %f", vName, float64(value), sum.Value)
			}
			return nil
		}
	}
	return fmt.Errorf("[%s]: could not find tags, wantTags: %s in rows %v", vName, wantTags, rows)
}

// tagsForReceiverView returns the tags that are needed for the receiver views.
func tagsForReceiverView(receiver config.ComponentID, transport string) []tag.Tag {
	tags := make([]tag.Tag, 0, 2)

	tags = append(tags, tag.Tag{Key: receiverTag, Value: receiver.String()})
	if transport != "" {
		tags = append(tags, tag.Tag{Key: transportTag, Value: transport})
	}

	return tags
}

// tagsForScraperView returns the tags that are needed for the scraper views.
func tagsForScraperView(receiver config.ComponentID, scraper config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: receiverTag, Value: receiver.String()},
		{Key: scraperTag, Value: scraper.String()},
	}
}

// tagsForProcessorView returns the tags that are needed for the processor views.
func tagsForProcessorView(processor config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: processorTag, Value: processor.String()},
	}
}

// tagsForExporterView returns the tags that are needed for the exporter views.
func tagsForExporterView(exporter config.ComponentID) []tag.Tag {
	return []tag.Tag{
		{Key: exporterTag, Value: exporter.String()},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}
