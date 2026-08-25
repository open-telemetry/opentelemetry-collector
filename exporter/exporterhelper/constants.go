// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"errors"
)

var (
	// errNilConfig is returned when an empty name is given.
	errNilConfig = errors.New("nil config")
	// errNilLogger is returned when a logger is nil
	errNilLogger = errors.New("nil logger")
	// errNilPushTraces is returned when a nil PushTraces is given.
	errNilPushTraces = errors.New("nil PushTraces")
	// errNilPushMetrics is returned when a nil PushMetrics is given.
	errNilPushMetrics = errors.New("nil PushMetrics")
	// errNilPushLogs is returned when a nil PushLogs is given.
	errNilPushLogs = errors.New("nil PushLogs")
)
