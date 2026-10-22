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

// ConsumeTraces takes pipeline ownership of the data and releases it once the downstream consumer
// returns. It is a no-op if the data is already owned by the pipeline.
func (c refTraces) ConsumeTraces(ctx context.Context, ld ptrace.Traces) error {
	if pref.MarkPipelineOwnedTraces(ld) {
		defer pref.UnrefTraces(ld)
	}
	return c.consumer.ConsumeTraces(ctx, ld)
}

func (c refTraces) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
