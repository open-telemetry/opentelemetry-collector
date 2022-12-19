// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
