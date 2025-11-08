package arc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkmetricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/pipeline"
)

func collectOne(t *testing.T, r *sdkmetric.ManualReader) sdkmetricdata.ResourceMetrics {
	t.Helper()
	var out sdkmetricdata.ResourceMetrics
	require.NoError(t, r.Collect(context.Background(), &out))
	return out
}

func findSum(rm sdkmetricdata.ResourceMetrics, name string, attrs []attribute.KeyValue) (bool, int64) {
	want := attribute.NewSet(attrs...) // canonical comparison set
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if s, ok := m.Data.(sdkmetricdata.Sum[int64]); ok {
				for _, dp := range s.DataPoints {
					// Works for both pointer and value receivers
					if (&dp.Attributes).Equals(&want) || dp.Attributes.Equals(&want) {
						return true, dp.Value
					}
				}
			}
		}
	}
	return false, 0
}

func TestController_BackpressureEmitsMetricsAndShrinks(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	ts := componenttest.NewNopTelemetrySettings()
	ts.MeterProvider = mp

	tel, err := metadata.NewTelemetryBuilder(ts)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.InitialLimit = 8
	cfg.MaxConcurrency = 16
	cfg.DecreaseRatio = 0.5

	id := component.MustNewID("otlp")
	ctrl := NewController(cfg, tel, id, pipeline.SignalTraces)

	// Initialize EWMA so the controller takes the 'normal' decrease path
	ctrl.mu.Lock()
	ctrl.st.past.update(0.01) // small mean to make tick scheduling trivial
	ctrl.st.limit = 8
	ctrl.st.hadPressure = true // force a decrease on adjust
	ctrl.mu.Unlock()

	// Trigger a window tick -> adjust() sees hadPressure => multiplicative decrease.
	ctrl.Feedback(context.Background(), 1*time.Millisecond, true /* success */, true /* backpressure */)
	time.Sleep(10 * time.Millisecond)

	rm := collectOne(t, reader)

	common := []attribute.KeyValue{
		attribute.String("exporter", id.String()),
		attribute.String("data_type", pipeline.SignalTraces.String()),
	}

	found, backoffs := findSum(rm, "otelcol_exporter_arc_backoff_events", common)
	require.True(t, found, "backoff_events not found")
	require.GreaterOrEqual(t, backoffs, int64(1))

	down := append(common, attribute.String("direction", "down"))
	found, downs := findSum(rm, "otelcol_exporter_arc_limit_changes", down)
	require.True(t, found, "limit_changes{down} not found")
	require.GreaterOrEqual(t, downs, int64(1))
}
