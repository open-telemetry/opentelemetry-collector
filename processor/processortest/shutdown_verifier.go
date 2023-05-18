// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/processor"
)

func verifyTracesDoesNotProduceAfterShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	// Create a proc and output its produce to a sink.
	nextSink := new(consumertest.TracesSink)
	proc, err := factory.CreateTracesProcessor(context.Background(), NewNopCreateSettings(), cfg, nextSink)
	if err != nil {
		if errors.Is(err, component.ErrDataTypeIsNotSupported) {
			return
		}
		require.NoError(t, err)
	}
	assert.NoError(t, proc.Start(context.Background(), componenttest.NewNopHost()))

	// Send some traces to the proc.
	const generatedCount = 10
	for i := 0; i < generatedCount; i++ {
		require.NoError(t, proc.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}

	// Now shutdown the proc.
	assert.NoError(t, proc.Shutdown(context.Background()))

	// The Shutdown() is done. It means the proc must have sent everything we
	// gave it to the next sink.
	assert.EqualValues(t, generatedCount, nextSink.SpanCount())
}

// VerifyShutdown verifies the processor doesn't produce telemetry data after shutdown.
func VerifyShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	verifyTracesDoesNotProduceAfterShutdown(t, factory, cfg)
	// TODO: add metrics and logs verification.
	// TODO: add other shutdown verifications.
}
