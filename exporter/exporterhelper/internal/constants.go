// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import "errors"

var (
	// errNilLogger is returned when a logger is nil
	errNilLogger = errors.New("nil logger")
	// errNilConsumeRequest is returned when a nil PushTraces is given.
	errNilConsumeRequest = errors.New("nil RequestConsumeFunc")
	// errNilTracesConverter is returned when a nil RequestFromTracesFunc is given.
	errNilTracesConverter = errors.New("nil RequestFromTracesFunc")
	// errNilMetricsConverter is returned when a nil RequestFromMetricsFunc is given.
	errNilMetricsConverter = errors.New("nil RequestFromMetricsFunc")
	// errNilLogsConverter is returned when a nil RequestFromLogsFunc is given.
	errNilLogsConverter = errors.New("nil RequestFromLogsFunc")
)
