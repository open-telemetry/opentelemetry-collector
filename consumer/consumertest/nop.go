// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewNop returns a Consumer that just drops all received data and returns no error.
func NewNop() Consumer {
	return &baseConsumer{
		ConsumeTracesFunc:  func(_ context.Context, _ ptrace.Traces) error { return nil },
		ConsumeMetricsFunc: func(_ context.Context, _ pmetric.Metrics) error { return nil },
		ConsumeLogsFunc:    func(_ context.Context, _ plog.Logs) error { return nil },
	}
}
