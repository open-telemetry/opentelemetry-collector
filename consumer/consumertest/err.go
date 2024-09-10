// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"
import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewErr returns a Consumer that just drops all received data and returns the specified error to Consume* callers.
func NewErr(err error) Consumer {
	return &baseConsumer{
		ConsumeTracesFunc:   func(context.Context, ptrace.Traces) error { return err },
		ConsumeMetricsFunc:  func(context.Context, pmetric.Metrics) error { return err },
		ConsumeLogsFunc:     func(context.Context, plog.Logs) error { return err },
		ConsumeProfilesFunc: func(context.Context, pprofile.Profiles) error { return err },
	}
}
