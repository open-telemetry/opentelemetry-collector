// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline // import "go.opentelemetry.io/collector/pipeline"

import (
	"go.opentelemetry.io/collector/internal/globalsignal"
)

// Signal represents the signals supported by the collector. We currently support
// collecting metrics, traces and logs, this can expand in the future.
type Signal = globalsignal.Signal

var (
	SignalTraces  = globalsignal.MustNewSignal("traces")
	SignalMetrics = globalsignal.MustNewSignal("metrics")
	SignalLogs    = globalsignal.MustNewSignal("logs")
)
