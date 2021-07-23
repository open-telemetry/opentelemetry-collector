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

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func verifyTracesProcessorDoesntProduceAfterShutdown(t *testing.T, factory component.ProcessorFactory, cfg config.Processor) {
	// Create a processor and output its produce to a sink.
	nextSink := new(consumertest.TracesSink)
	processor, err := factory.CreateTracesProcessor(
		context.Background(),
		NewNopProcessorCreateSettings(),
		cfg,
		nextSink,
	)
	if err != nil {
		if err == componenterror.ErrDataTypeIsNotSupported {
			return
		}
		require.NoError(t, err)
	}
	err = processor.Start(context.Background(), NewNopHost())
	assert.NoError(t, err)

	// Send some traces to the processor.
	const generatedCount = 10
	for i := 0; i < generatedCount; i++ {
		require.NoError(t, processor.ConsumeTraces(context.Background(), testdata.GenerateTracesOneSpan()))
	}

	// Now shutdown the processor.
	err = processor.Shutdown(context.Background())
	assert.NoError(t, err)

	// The Shutdown() is done. It means the processor must have sent everything we
	// gave it to the next sink.
	assert.EqualValues(t, generatedCount, nextSink.SpanCount())
}

// VerifyProcessorShutdown verifies the processor doesn't produce telemetry data after shutdown.
func VerifyProcessorShutdown(t *testing.T, factory component.ProcessorFactory, cfg config.Processor) {
	verifyTracesProcessorDoesntProduceAfterShutdown(t, factory, cfg)
	// TODO: add metrics and logs verification.
	// TODO: add other shutdown verifications.
}
