package ctracetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTraces(1)
	want := make([]ptrace.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllTraces()))
	assert.Equal(t, 0, sink.SpanCount())
}
