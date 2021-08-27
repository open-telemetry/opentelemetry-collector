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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

func TestDefaultReceivers(t *testing.T) {
	allFactories, err := Components()
	assert.NoError(t, err)

	rcvrFactories := allFactories.Receivers

	tests := []struct {
		receiver    config.Type
		getConfigFn getReceiverConfigFn
	}{
		{
			receiver: "otlp",
			getConfigFn: func() config.Receiver {
				cfg := otlpreceiver.NewFactory().CreateDefaultConfig()
				cfg.(*otlpreceiver.Config).GRPC.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)
				cfg.(*otlpreceiver.Config).HTTP.Endpoint = testutil.GetAvailableLocalAddress(t)
				return cfg
			},
		},
	}

	assert.Equal(t, len(tests), len(rcvrFactories))
	for _, tt := range tests {
		t.Run(string(tt.receiver), func(t *testing.T) {
			factory, ok := rcvrFactories[tt.receiver]
			require.True(t, ok)
			assert.Equal(t, tt.receiver, factory.Type())
			assert.Equal(t, config.NewID(tt.receiver), factory.CreateDefaultConfig().ID())

			verifyReceiverLifecycle(t, factory, tt.getConfigFn)
		})
	}
}

// getReceiverConfigFn is used customize the configuration passed to the verification.
// This is used to change ports or provide values required but not provided by the
// default configuration.
type getReceiverConfigFn func() config.Receiver

// verifyReceiverLifecycle is used to test if a receiver type can handle the typical
// lifecycle of a component. The getConfigFn parameter only need to be specified if
// the test can't be done with the default configuration for the component.
func verifyReceiverLifecycle(t *testing.T, factory component.ReceiverFactory, getConfigFn getReceiverConfigFn) {
	ctx := context.Background()
	host := newAssertNoErrorHost(t)
	receiverCreateSet := componenttest.NewNopReceiverCreateSettings()

	if getConfigFn == nil {
		getConfigFn = factory.CreateDefaultConfig
	}

	createFns := []createReceiverFn{
		wrapCreateLogsRcvr(factory),
		wrapCreateTracesRcvr(factory),
		wrapCreateMetricsRcvr(factory),
	}

	for _, createFn := range createFns {
		firstRcvr, err := createFn(ctx, receiverCreateSet, getConfigFn())
		if errors.Is(err, componenterror.ErrDataTypeIsNotSupported) {
			continue
		}
		require.NoError(t, err)
		require.NoError(t, firstRcvr.Start(ctx, host))
		require.NoError(t, firstRcvr.Shutdown(ctx))

		secondRcvr, err := createFn(ctx, receiverCreateSet, getConfigFn())
		require.NoError(t, err)
		require.NoError(t, secondRcvr.Start(ctx, host))
		require.NoError(t, secondRcvr.Shutdown(ctx))
	}
}

// assertNoErrorHost implements a component.Host that asserts that there were no errors.
type createReceiverFn func(
	ctx context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
) (component.Receiver, error)

func wrapCreateLogsRcvr(factory component.ReceiverFactory) createReceiverFn {
	return func(ctx context.Context, set component.ReceiverCreateSettings, cfg config.Receiver) (component.Receiver, error) {
		return factory.CreateLogsReceiver(ctx, set, cfg, consumertest.NewNop())
	}
}

func wrapCreateMetricsRcvr(factory component.ReceiverFactory) createReceiverFn {
	return func(ctx context.Context, set component.ReceiverCreateSettings, cfg config.Receiver) (component.Receiver, error) {
		return factory.CreateMetricsReceiver(ctx, set, cfg, consumertest.NewNop())
	}
}

func wrapCreateTracesRcvr(factory component.ReceiverFactory) createReceiverFn {
	return func(ctx context.Context, set component.ReceiverCreateSettings, cfg config.Receiver) (component.Receiver, error) {
		return factory.CreateTracesReceiver(ctx, set, cfg, consumertest.NewNop())
	}
}
