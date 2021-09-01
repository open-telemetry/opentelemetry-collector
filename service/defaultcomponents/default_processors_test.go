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

package defaultcomponents

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/memorylimiter"
)

func TestDefaultProcessors(t *testing.T) {
	allFactories, err := Components()
	require.NoError(t, err)

	procFactories := allFactories.Processors

	tests := []struct {
		processor   config.Type
		getConfigFn getProcessorConfigFn
	}{
		{
			processor: "batch",
		},
		{
			processor: "memory_limiter",
			getConfigFn: func() config.Processor {
				cfg := procFactories["memory_limiter"].CreateDefaultConfig().(*memorylimiter.Config)
				cfg.CheckInterval = 100 * time.Millisecond
				cfg.MemoryLimitMiB = 1024 * 1024
				return cfg
			},
		},
	}

	assert.Equal(t, len(tests), len(procFactories))
	for _, tt := range tests {
		t.Run(string(tt.processor), func(t *testing.T) {
			factory, ok := procFactories[tt.processor]
			require.True(t, ok)
			assert.Equal(t, tt.processor, factory.Type())
			assert.EqualValues(t, config.NewID(tt.processor), factory.CreateDefaultConfig().ID())

			verifyProcessorLifecycle(t, factory, tt.getConfigFn)
		})
	}
}

// getProcessorConfigFn is used customize the configuration passed to the verification.
// This is used to change ports or provide values required but not provided by the
// default configuration.
type getProcessorConfigFn func() config.Processor

// verifyProcessorLifecycle is used to test if an processor type can handle the typical
// lifecycle of a component. The getConfigFn parameter only need to be specified if
// the test can't be done with the default configuration for the component.
func verifyProcessorLifecycle(t *testing.T, factory component.ProcessorFactory, getConfigFn getProcessorConfigFn) {
	ctx := context.Background()
	host := newAssertNoErrorHost(t)
	processorCreationSet := componenttest.NewNopProcessorCreateSettings()

	if getConfigFn == nil {
		getConfigFn = factory.CreateDefaultConfig
	}

	createFns := []createProcessorFn{
		wrapCreateLogsProc(factory),
		wrapCreateTracesProc(factory),
		wrapCreateMetricsProc(factory),
	}

	for _, createFn := range createFns {
		firstExp, err := createFn(ctx, processorCreationSet, getConfigFn())
		if errors.Is(err, componenterror.ErrDataTypeIsNotSupported) {
			continue
		}
		require.NoError(t, err)
		require.NoError(t, firstExp.Start(ctx, host))
		require.NoError(t, firstExp.Shutdown(ctx))

		secondExp, err := createFn(ctx, processorCreationSet, getConfigFn())
		require.NoError(t, err)
		require.NoError(t, secondExp.Start(ctx, host))
		require.NoError(t, secondExp.Shutdown(ctx))
	}
}

type createProcessorFn func(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
) (component.Processor, error)

func wrapCreateLogsProc(factory component.ProcessorFactory) createProcessorFn {
	return func(ctx context.Context, set component.ProcessorCreateSettings, cfg config.Processor) (component.Processor, error) {
		return factory.CreateLogsProcessor(ctx, set, cfg, consumertest.NewNop())
	}
}

func wrapCreateMetricsProc(factory component.ProcessorFactory) createProcessorFn {
	return func(ctx context.Context, set component.ProcessorCreateSettings, cfg config.Processor) (component.Processor, error) {
		return factory.CreateMetricsProcessor(ctx, set, cfg, consumertest.NewNop())
	}
}

func wrapCreateTracesProc(factory component.ProcessorFactory) createProcessorFn {
	return func(ctx context.Context, set component.ProcessorCreateSettings, cfg config.Processor) (component.Processor, error) {
		return factory.CreateTracesProcessor(ctx, set, cfg, consumertest.NewNop())
	}
}
