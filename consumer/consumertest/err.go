// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"
import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewErr returns a Consumer that just drops all received data and returns the specified error to Consume* callers.
func NewErr(err error) Consumer {
	return &baseConsumer{
		ConsumeTracesFunc:  func(ctx context.Context, td ptrace.Traces) error { return err },
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error { return err },
		ConsumeLogsFunc:    func(ctx context.Context, ld plog.Logs) error { return err },
	}
}
