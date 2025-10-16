// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package refconsumer // import "go.opentelemetry.io/collector/service/internal/refconsumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func NewTraces(cons consumer.Traces) consumer.Traces {
	return refTraces{
		consumer: cons,
	}
}

type refTraces struct {
	consumer consumer.Traces
}

// ConsumeTraces measures telemetry before calling ConsumeTraces because the data may be mutated downstream
func (c refTraces) ConsumeTraces(ctx context.Context, ld ptrace.Traces) error {
	if pref.MarkPipelineOwnedTraces(ld) {
		defer pref.UnrefTraces(ld)
	}
	return c.consumer.ConsumeTraces(ctx, ld)
}

func (c refTraces) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
