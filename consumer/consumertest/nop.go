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

// NewNop returns a Consumer that just drops all received data and returns no error.
func NewNop() Consumer {
	return &baseConsumer{
		ConsumeTracesFunc:   func(context.Context, ptrace.Traces) error { return nil },
		ConsumeMetricsFunc:  func(context.Context, pmetric.Metrics) error { return nil },
		ConsumeLogsFunc:     func(context.Context, plog.Logs) error { return nil },
		ConsumeProfilesFunc: func(context.Context, pprofile.Profiles) error { return nil },
	}
}
