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

package loggingexporter

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/otlptext"
	"go.opentelemetry.io/collector/model/pdata"
)

type loggingExporter struct {
	logger           *zap.Logger
	debug            bool
	logsMarshaler    pdata.LogsMarshaler
	metricsMarshaler pdata.MetricsMarshaler
	tracesMarshaler  pdata.TracesMarshaler
}

func (s *loggingExporter) pushTraces(_ context.Context, td pdata.Traces) error {
	s.logger.Info("TracesExporter", zap.Int("#spans", td.SpanCount()))

	if !s.debug {
		return nil
	}

	buf, err := s.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return err
	}
	s.logger.Debug(string(buf))
	return nil
}

func (s *loggingExporter) pushMetrics(_ context.Context, md pdata.Metrics) error {
	s.logger.Info("MetricsExporter", zap.Int("#metrics", md.MetricCount()))

	if !s.debug {
		return nil
	}

	buf, err := s.metricsMarshaler.MarshalMetrics(md)
	if err != nil {
		return err
	}
	s.logger.Debug(string(buf))
	return nil
}

func (s *loggingExporter) pushLogs(_ context.Context, ld pdata.Logs) error {
	s.logger.Info("LogsExporter", zap.Int("#logs", ld.LogRecordCount()))

	if !s.debug {
		return nil
	}

	buf, err := s.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return err
	}
	s.logger.Debug(string(buf))
	return nil
}

func newLoggingExporter(level string, logger *zap.Logger) *loggingExporter {
	return &loggingExporter{
		debug:            strings.ToLower(level) == "debug",
		logger:           logger,
		logsMarshaler:    otlptext.NewTextLogsMarshaler(),
		metricsMarshaler: otlptext.NewTextMetricsMarshaler(),
		tracesMarshaler:  otlptext.NewTextTracesMarshaler(),
	}
}

// newTracesExporter creates an exporter.TracesExporter that just drops the
// received data and logs debugging messages.
func newTracesExporter(config config.Exporter, level string, logger *zap.Logger, set component.ExporterCreateSettings) (component.TracesExporter, error) {
	s := newLoggingExporter(level, logger)
	return exporterhelper.NewTracesExporter(
		config,
		set,
		s.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(logger)),
	)
}

// newMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func newMetricsExporter(config config.Exporter, level string, logger *zap.Logger, set component.ExporterCreateSettings) (component.MetricsExporter, error) {
	s := newLoggingExporter(level, logger)
	return exporterhelper.NewMetricsExporter(
		config,
		set,
		s.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(logger)),
	)
}

// newLogsExporter creates an exporter.LogsExporter that just drops the
// received data and logs debugging messages.
func newLogsExporter(config config.Exporter, level string, logger *zap.Logger, set component.ExporterCreateSettings) (component.LogsExporter, error) {
	s := newLoggingExporter(level, logger)
	return exporterhelper.NewLogsExporter(
		config,
		set,
		s.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(logger)),
	)
}

func loggerSync(logger *zap.Logger) func(context.Context) error {
	return func(context.Context) error {
		// Currently Sync() return a different error depending on the OS.
		// Since these are not actionable ignore them.
		err := logger.Sync()
		if osErr, ok := err.(*os.PathError); ok {
			wrappedErr := osErr.Unwrap()
			if knownSyncError(wrappedErr) {
				err = nil
			}
		}
		return err
	}
}
