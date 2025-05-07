// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
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
}

func newDebugExporter(logger *zap.Logger, verbosity configtelemetry.Level) *debugExporter {
	var logsMarshaler plog.Marshaler
	var metricsMarshaler pmetric.Marshaler
	var tracesMarshaler ptrace.Marshaler
	var profilesMarshaler pprofile.Marshaler
	if verbosity == configtelemetry.LevelDetailed {
		logsMarshaler = otlptext.NewTextLogsMarshaler()
		metricsMarshaler = otlptext.NewTextMetricsMarshaler()
		tracesMarshaler = otlptext.NewTextTracesMarshaler()
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
	}
}

func (s *debugExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
	s.logger.Info("Traces",
		zap.Int("resource spans", td.ResourceSpans().Len()),
		zap.Int("spans", td.SpanCount()))
	if s.verbosity == configtelemetry.LevelBasic {
		return nil
	}

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
