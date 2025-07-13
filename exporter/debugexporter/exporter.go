// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"
	"slices"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type debugExporter struct {
	verbosity         configtelemetry.Level
	logger            *zap.Logger
	logsMarshaler     plog.Marshaler
	metricsMarshaler  pmetric.Marshaler
	tracesMarshaler   ptrace.Marshaler
	profilesMarshaler pprofile.Marshaler
	outputConfig      internal.OutputConfig
}

func newDebugExporter(logger *zap.Logger, verbosity configtelemetry.Level, outputConfig internal.OutputConfig) *debugExporter {
	var logsMarshaler plog.Marshaler
	var metricsMarshaler pmetric.Marshaler
	var tracesMarshaler ptrace.Marshaler
	var profilesMarshaler pprofile.Marshaler
	if verbosity == configtelemetry.LevelDetailed {
		logsMarshaler = otlptext.NewTextLogsMarshaler(outputConfig)
		metricsMarshaler = otlptext.NewTextMetricsMarshaler(outputConfig)
		tracesMarshaler = otlptext.NewTextTracesMarshaler(outputConfig)
		profilesMarshaler = otlptext.NewTextProfilesMarshaler()
	} else {
		logsMarshaler = normal.NewNormalLogsMarshaler()
		metricsMarshaler = normal.NewNormalMetricsMarshaler()
		tracesMarshaler = normal.NewNormalTracesMarshaler()
		profilesMarshaler = normal.NewNormalProfilesMarshaler()
	}
	return &debugExporter{
		verbosity:         verbosity,
		logger:            logger,
		logsMarshaler:     logsMarshaler,
		metricsMarshaler:  metricsMarshaler,
		tracesMarshaler:   tracesMarshaler,
		profilesMarshaler: profilesMarshaler,
		outputConfig:      outputConfig,
	}
}

func matchAttributes(key string, outputConfig internal.OutputConfig) bool {
	if len(outputConfig.Record.AttributesOutputConfig.Include) == 0 {
		return true
	}
	return slices.Contains(outputConfig.Record.AttributesOutputConfig.Include, key)
}

func (s *debugExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
	s.logger.Info("Traces",
		zap.Int("resource spans", td.ResourceSpans().Len()),
		zap.Int("spans", td.SpanCount()))
	if s.verbosity == configtelemetry.LevelBasic {
		return nil
	}

	//for _, rs := range td.ResourceSpans().All() {
	//	rs.Resource().Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//		return !matchAttributes(k, s.outputConfig)
	//	})
	//	for _, ss := range rs.ScopeSpans().All() {
	//		ss.Scope().Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//			return !matchAttributes(k, s.outputConfig)
	//		})
	//		for _, span := range ss.Spans().All() {
	//			span.Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//				return !matchAttributes(k, s.outputConfig)
	//			})
	//		}
	//	}
	//}

	buf, err := s.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}

func (s *debugExporter) pushMetrics(_ context.Context, md pmetric.Metrics) error {
	s.logger.Info("Metrics",
		zap.Int("resource metrics", md.ResourceMetrics().Len()),
		zap.Int("metrics", md.MetricCount()),
		zap.Int("data points", md.DataPointCount()))
	if s.verbosity == configtelemetry.LevelBasic {
		return nil
	}

	//for _, rm := range md.ResourceMetrics().All() {
	//	rm.Resource().Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//		return !matchAttributes(k, s.outputConfig)
	//	})
	//	for _, sm := range rm.ScopeMetrics().All() {
	//		sm.Scope().Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//			return !matchAttributes(k, s.outputConfig)
	//		})
	//	}
	//}

	buf, err := s.metricsMarshaler.MarshalMetrics(md)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}

func (s *debugExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	s.logger.Info("Logs",
		zap.Int("resource logs", ld.ResourceLogs().Len()),
		zap.Int("log records", ld.LogRecordCount()))

	if s.verbosity == configtelemetry.LevelBasic {
		return nil
	}

	//for _, resourceLog := range ld.ResourceLogs().All() {
	//	resourceLog.Resource().Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//		return !matchAttributes(k, s.outputConfig)
	//	})
	//	for _, scopeLog := range resourceLog.ScopeLogs().All() {
	//		for _, logRecord := range scopeLog.LogRecords().All() {
	//			logRecord.Attributes().RemoveIf(func(k string, _ pcommon.Value) bool {
	//				return !matchAttributes(k, s.outputConfig)
	//			})
	//		}
	//	}
	//}

	buf, err := s.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}

func (s *debugExporter) pushProfiles(_ context.Context, pd pprofile.Profiles) error {
	s.logger.Info("Profiles",
		zap.Int("resource profiles", pd.ResourceProfiles().Len()),
		zap.Int("sample records", pd.SampleCount()))

	if s.verbosity == configtelemetry.LevelBasic {
		return nil
	}

	buf, err := s.profilesMarshaler.MarshalProfiles(pd)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}
