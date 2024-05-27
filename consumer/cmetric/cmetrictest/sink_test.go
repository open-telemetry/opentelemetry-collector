package cmetrictest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMetricsSink(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetrics(1)
	want := make([]pmetric.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetrics(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllMetrics()))
	assert.Equal(t, 0, sink.DataPointCount())
}
