// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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

// TestProcessorSpans verifies the embedded exporterhelper spans are reframed as
// processor spans: reported under this processor's scope with processor/-prefixed
// names, with parents and batch links preserved.
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
		switch s.Name() {
		case "processor/enqueue":
			enqueue++
		case exportSpanName:
			export++
			require.Len(t, s.Links(), 2, "batch links should be preserved")
		}
	}
	require.Equal(t, 2, enqueue)
	require.Equal(t, 1, export)
}
