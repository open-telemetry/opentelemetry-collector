// Code generated by mdatagen. DO NOT EDIT.

package nopreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestComponentLifecycle(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name     string
		createFn func(ctx context.Context, set receiver.CreateSettings, cfg component.Config) (component.Component, error)
	}{

		{
			name: "logs",
			createFn: func(ctx context.Context, set receiver.CreateSettings, cfg component.Config) (component.Component, error) {
				return factory.CreateLogsReceiver(ctx, set, cfg, consumertest.NewNop())
			},
		},

		{
			name: "metrics",
			createFn: func(ctx context.Context, set receiver.CreateSettings, cfg component.Config) (component.Component, error) {
				return factory.CreateMetricsReceiver(ctx, set, cfg, consumertest.NewNop())
			},
		},

		{
			name: "traces",
			createFn: func(ctx context.Context, set receiver.CreateSettings, cfg component.Config) (component.Component, error) {
				return factory.CreateTracesReceiver(ctx, set, cfg, consumertest.NewNop())
			},
		},
	}

	cm, err := confmaptest.LoadConf("metadata.yaml")
	require.NoError(t, err)
	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub("tests::config")
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(&cfg))

	for _, test := range tests {
		t.Run(test.name+"-shutdown", func(t *testing.T) {
			c, err := test.createFn(context.Background(), receivertest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			err = c.Shutdown(context.Background())
			require.NoError(t, err)
		})
		t.Run(test.name+"-lifecycle", func(t *testing.T) {
			firstRcvr, err := test.createFn(context.Background(), receivertest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			host := componenttest.NewNopHost()
			require.NoError(t, err)
			require.NoError(t, firstRcvr.Start(context.Background(), host))
			require.NoError(t, firstRcvr.Shutdown(context.Background()))
			secondRcvr, err := test.createFn(context.Background(), receivertest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			require.NoError(t, secondRcvr.Start(context.Background(), host))
			require.NoError(t, secondRcvr.Shutdown(context.Background()))
		})
	}
}
