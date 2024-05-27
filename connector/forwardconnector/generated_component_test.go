// Code generated by mdatagen. DO NOT EDIT.

package forwardconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer/cmetric"
	"go.opentelemetry.io/collector/consumer/conslog"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/ctrace"
)

func TestComponentFactoryType(t *testing.T) {
	require.Equal(t, "forward", NewFactory().Type().String())
}

func TestComponentConfigStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(NewFactory().CreateDefaultConfig()))
}

func TestComponentLifecycle(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name     string
		createFn func(ctx context.Context, set connector.CreateSettings, cfg component.Config) (component.Component, error)
	}{

		{
			name: "logs_to_logs",
			createFn: func(ctx context.Context, set connector.CreateSettings, cfg component.Config) (component.Component, error) {
				router := connector.NewLogsRouter(map[component.ID]conslog.Logs{component.NewID(component.DataTypeLogs): consumertest.NewNop()})
				return factory.CreateLogsToLogs(ctx, set, cfg, router)
			},
		},

		{
			name: "metrics_to_metrics",
			createFn: func(ctx context.Context, set connector.CreateSettings, cfg component.Config) (component.Component, error) {
				router := connector.NewMetricsRouter(map[component.ID]cmetric.Metrics{component.NewID(component.DataTypeMetrics): consumertest.NewNop()})
				return factory.CreateMetricsToMetrics(ctx, set, cfg, router)
			},
		},

		{
			name: "traces_to_traces",
			createFn: func(ctx context.Context, set connector.CreateSettings, cfg component.Config) (component.Component, error) {
				router := connector.NewTracesRouter(map[component.ID]ctrace.Traces{component.NewID(component.DataTypeTraces): consumertest.NewNop()})
				return factory.CreateTracesToTraces(ctx, set, cfg, router)
			},
		},
	}

	cm, err := confmaptest.LoadConf("metadata.yaml")
	require.NoError(t, err)
	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub("tests::config")
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))

	for _, test := range tests {
		t.Run(test.name+"-shutdown", func(t *testing.T) {
			c, err := test.createFn(context.Background(), connectortest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			err = c.Shutdown(context.Background())
			require.NoError(t, err)
		})
		t.Run(test.name+"-lifecycle", func(t *testing.T) {
			firstConnector, err := test.createFn(context.Background(), connectortest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			host := componenttest.NewNopHost()
			require.NoError(t, err)
			require.NoError(t, firstConnector.Start(context.Background(), host))
			require.NoError(t, firstConnector.Shutdown(context.Background()))
			secondConnector, err := test.createFn(context.Background(), connectortest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)
			require.NoError(t, secondConnector.Start(context.Background(), host))
			require.NoError(t, secondConnector.Shutdown(context.Background()))
		})
	}
}
