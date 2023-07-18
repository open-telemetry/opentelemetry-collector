// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jsonloggingexporter // import "go.opentelemetry.io/collector/exporter/jsonloggingexporter"

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/jsonloggingexporter/internal/otlptext"
	"go.opentelemetry.io/collector/pdata/plog"
)

type jsonLoggingExporter struct {
	verbosity     configtelemetry.Level
	logger        *zap.Logger
	logsMarshaler plog.Marshaler
}

func (s *jsonLoggingExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	s.logger.Info("LogsExporter",
		zap.Int("resource logs", ld.ResourceLogs().Len()),
		zap.Int("log records", ld.LogRecordCount()))
	if s.verbosity != configtelemetry.LevelDetailed {
		return nil
	}

	buf, err := s.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}

func newLoggingExporter(logger *zap.Logger, verbosity configtelemetry.Level) *jsonLoggingExporter {
	return &jsonLoggingExporter{
		verbosity:     verbosity,
		logger:        logger,
		logsMarshaler: otlptext.NewJsonLogsMarshaler(),
	}
}

func loggerSync(logger *zap.Logger) func(context.Context) error {
	return func(context.Context) error {
		// Currently Sync() return a different error depending on the OS.
		// Since these are not actionable ignore them.
		err := logger.Sync()
		osErr := &os.PathError{}
		if errors.As(err, &osErr) {
			wrappedErr := osErr.Unwrap()
			if knownSyncError(wrappedErr) {
				err = nil
			}
		}
		return err
	}
}
