// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package refconsumer // import "go.opentelemetry.io/collector/service/internal/refconsumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func NewLogs(cons consumer.Logs) consumer.Logs {
	return refLogs{
		consumer: cons,
	}
}

type refLogs struct {
	consumer consumer.Logs
}

// ConsumeLogs measures telemetry before calling ConsumeLogs because the data may be mutated downstream
func (c refLogs) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	if pref.MarkPipelineOwnedLogs(ld) {
		defer pref.UnrefLogs(ld)
	}
	return c.consumer.ConsumeLogs(ctx, ld)
}

func (c refLogs) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
