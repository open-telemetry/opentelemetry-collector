// Code generated by mdatagen. DO NOT EDIT.

package forwardconnector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
)

var typ = component.MustNewType("forward")

func TestComponentFactoryType(t *testing.T) {
	require.Equal(t, typ, NewFactory().Type())
}

func TestComponentConfigStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(NewFactory().CreateDefaultConfig()))
}

func TestComponentLifecycle(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		createFn func(ctx context.Context, set connector.Settings, cfg component.Config) (component.Component, error)
		name     string
	}{

		{
			name: "logs_to_logs",
			createFn: func(ctx context.Context, set connector.Settings, cfg component.Config) (component.Component, error) {
				router := connector.NewLogsRouter(map[pipeline.ID]consumer.Logs{pipeline.NewID(pipeline.SignalLogs): consumertest.NewNop()})
				return factory.CreateLogsToLogs(ctx, set, cfg, router)
			},
		},

		{
			name: "metrics_to_metrics",
			createFn: func(ctx context.Context, set connector.Settings, cfg component.Config) (component.Component, error) {
				router := connector.NewMetricsRouter(map[pipeline.ID]consumer.Metrics{pipeline.NewID(pipeline.SignalMetrics): consumertest.NewNop()})
				return factory.CreateMetricsToMetrics(ctx, set, cfg, router)
			},
		},

		{
			name: "traces_to_traces",
			createFn: func(ctx context.Context, set connector.Settings, cfg component.Config) (component.Component, error) {
				router := connector.NewTracesRouter(map[pipeline.ID]consumer.Traces{pipeline.NewID(pipeline.SignalTraces): consumertest.NewNop()})
				return factory.CreateTracesToTraces(ctx, set, cfg, router)
			},
		},
	}

	cm, err := confmaptest.LoadConf("metadata.yaml")
	require.NoError(t, err)
	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub("tests::config")
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(&cfg))

	envMap, err := cm.Sub("tests::env")
	require.NoError(t, err)
	for key, value := range envMap.ToStringMap() {
		t.Setenv(key, fmt.Sprintf("%v", value))
	}

	for _, tt := range tests {
		t.Run(tt.name+"-shutdown", func(t *testing.T) {
			c, err := tt.createFn(context.Background(), connectortest.NewNopSettings(typ), cfg)
			require.NoError(t, err)
			err = c.Shutdown(context.Background())
			require.NoError(t, err)
		})
		t.Run(tt.name+"-lifecycle", func(t *testing.T) {
			firstConnector, err := tt.createFn(context.Background(), connectortest.NewNopSettings(typ), cfg)
			require.NoError(t, err)
			host := componenttest.NewNopHost()
			require.NoError(t, err)
			require.NoError(t, firstConnector.Start(context.Background(), host))
			require.NoError(t, firstConnector.Shutdown(context.Background()))
			secondConnector, err := tt.createFn(context.Background(), connectortest.NewNopSettings(typ), cfg)
			require.NoError(t, err)
			require.NoError(t, secondConnector.Start(context.Background(), host))
			require.NoError(t, secondConnector.Shutdown(context.Background()))
		})
	}
}
