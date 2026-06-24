// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

func TestRenameSpan(t *testing.T) {
	require.Equal(t, "processor/enqueue", renameSpan("exporter/enqueue"))
	require.Equal(t, "processor/queuebatch/traces", renameSpan("exporter/queuebatch/traces"))
	require.Equal(t, "unrelated", renameSpan("unrelated"))
}

func TestRemapSpanAttributes(t *testing.T) {
	in := []attribute.KeyValue{
		attribute.String("exporter", "queuebatch"),
		attribute.String("data_type", "traces"),
		attribute.Int64("items.sent", 5),
	}
	out := remapSpanAttributes(in)
	require.Equal(t, []attribute.KeyValue{
		attribute.String(processorKey, "queuebatch"),
		attribute.String(signalKey, "traces"),
		attribute.Int64("items.sent", 5),
	}, out)
}

// TestProcessorSpans verifies the embedded exporterhelper spans are reframed as
// processor spans: processor/-namespaced names, this processor's scope, and
// processor/otel.signal attributes, with batch links preserved.
func TestProcessorSpans(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	// Accept asynchronously and do not flush by size, so the two requests merge
	// into a single batch flushed on shutdown, producing one linked export span.
	cfg.WaitForResult = false
	cfg.Batch.Get().Sizer = exporterhelper.RequestSizerTypeItems
	cfg.Batch.Get().MinSize = 1000

	p, err := newTracesProcessor(context.Background(), set, cfg, new(consumertest.TracesSink))
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(2)))
	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(2)))
	require.NoError(t, p.Shutdown(context.Background()))

	exportSpanName := fmt.Sprintf("processor/%s/%s", set.ID.String(), pipeline.SignalTraces.String())
	var enqueue, export int
	for _, s := range tt.SpanRecorder.Ended() {
		require.Equal(t, metadata.ScopeName, s.InstrumentationScope().Name)
		require.False(t, strings.HasPrefix(s.Name(), exporterSpanPrefix), "span still in exporter namespace: %s", s.Name())
		for _, a := range s.Attributes() {
			require.NotEqual(t, attribute.Key(exporterAttrKey), a.Key)
			require.NotEqual(t, attribute.Key(dataTypeAttrKey), a.Key)
		}
		switch s.Name() {
		case "processor/enqueue":
			enqueue++
		case exportSpanName:
			export++
			require.Len(t, s.Links(), 2, "batch links should be preserved")
			require.Equal(t, set.ID.String(), spanAttr(s.Attributes(), processorKey))
			require.Equal(t, pipeline.SignalTraces.String(), spanAttr(s.Attributes(), signalKey))
		}
	}
	require.Equal(t, 2, enqueue)
	require.Equal(t, 1, export)
}

func spanAttr(attrs []attribute.KeyValue, key string) string {
	for _, a := range attrs {
		if a.Key == attribute.Key(key) {
			return a.Value.AsString()
		}
	}
	return ""
}
