// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

func newRightSizedTracesPipeline(tb testing.TB, cfg Config, stages int, push consumer.ConsumeTracesFunc) consumer.Traces {
	tb.Helper()
	ctx := context.Background()
	host := componenttest.NewNopHost()
	set := exporter.Settings{
		ID:                component.MustNewID("nop"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
	exp, err := exporterhelper.NewTraces(ctx, set, &cfg, push,
		exporterhelper.WithQueue(configoptional.Some(cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}))
	require.NoError(tb, err)
	require.NoError(tb, exp.Start(ctx, host))
	tb.Cleanup(func() { require.NoError(tb, exp.Shutdown(ctx)) })

	var next consumer.Traces = exp
	for i := 1; i < stages; i++ {
		set := processortest.NewNopSettings(metadata.Type)
		set.ID = component.NewIDWithName(metadata.Type, fmt.Sprintf("stage%d", i))
		p, err := newTracesProcessor(ctx, set, &cfg, next)
		require.NoError(tb, err)
		require.NoError(tb, p.Start(ctx, host))
		tb.Cleanup(func() { require.NoError(tb, p.Shutdown(ctx)) })
		next = p
	}
	return next
}

func TestRightSizedBatchingStages(t *testing.T) {
	marshaler := &ptrace.ProtoMarshaler{}
	for _, sizer := range []exporterhelper.RequestSizerType{
		exporterhelper.RequestSizerTypeItems,
		exporterhelper.RequestSizerTypeBytes,
	} {
		for _, bounded := range []bool{false, true} {
			for _, waitForResult := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/max_size=%t/wait_for_result=%t", sizer, bounded, waitForResult), func(t *testing.T) {
					cfg := *createDefaultConfig().(*Config)
					cfg.WaitForResult = waitForResult
					cfg.Batch.Get().FlushTimeout = time.Hour
					cfg.Batch.Get().Sizer = sizer
					cfg.Batch.Get().MinSize = 64
					if sizer == exporterhelper.RequestSizerTypeBytes {
						cfg.Batch.Get().MinSize = int64(marshaler.TracesSize(generateTraces(64)))
					}
					cfg.Batch.Get().MaxSize = 0
					if bounded {
						cfg.Batch.Get().MaxSize = 128
						if sizer == exporterhelper.RequestSizerTypeBytes {
							cfg.Batch.Get().MaxSize = int64(marshaler.TracesSize(generateTraces(128)))
						}
					}

					delivered := make(chan ptrace.Traces, 16)
					pipeline := newRightSizedTracesPipeline(t, cfg, 3, func(_ context.Context, td ptrace.Traces) error {
						delivered <- td
						return nil
					})
					for _, count := range []int{64, 96, 128} {
						input := generateTraces(count)
						t.Cleanup(func() { pref.UnrefTraces(input) })
						input.MarkReadOnly()
						want, err := marshaler.MarshalTraces(input)
						require.NoError(t, err)
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						done := make(chan error, 1)
						go func() { done <- pipeline.ConsumeTraces(ctx, input) }()
						var got ptrace.Traces
						select {
						case got = <-delivered:
						case <-ctx.Done():
							t.Fatal("right-sized request did not pass through before the flush timeout")
						}
						select {
						case consumeErr := <-done:
							require.NoError(t, consumeErr)
						case <-ctx.Done():
							t.Fatal("ConsumeTraces did not complete after delivery")
						}
						require.True(t, input == got, "right-sized requests must not be copied or split") //nolint:testifylint // Compare pdata identity, not contents.
						actual, err := marshaler.MarshalTraces(got)
						require.NoError(t, err)
						require.Equal(t, want, actual)
					}
				})
			}
		}
	}
}

func BenchmarkRightSizedBatchingStages(b *testing.B) {
	for _, stages := range []int{1, 3} {
		for _, mode := range []string{"queue_only", "min_size", "min_and_max_size"} {
			for _, count := range []int{64, 8192} {
				b.Run(fmt.Sprintf("%s/stages=%d/items=%d", mode, stages, count), func(b *testing.B) {
					cfg := *createDefaultConfig().(*Config)
					cfg.WaitForResult = true
					if mode == "queue_only" {
						cfg.Batch = configoptional.None[exporterhelper.BatchConfig]()
					} else {
						cfg.Batch.Get().FlushTimeout = time.Hour
						cfg.Batch.Get().MinSize = int64(count)
						if mode == "min_and_max_size" {
							cfg.Batch.Get().MaxSize = int64(count)
						}
					}
					delivered := 0
					pipeline := newRightSizedTracesPipeline(b, cfg, stages, func(context.Context, ptrace.Traces) error {
						delivered++
						return nil
					})
					input := generateTraces(count)
					b.Cleanup(func() { pref.UnrefTraces(input) })
					input.MarkReadOnly()
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						if err := pipeline.ConsumeTraces(context.Background(), input); err != nil {
							b.Fatal(err)
						}
					}
					require.Equal(b, b.N, delivered)
				})
			}
		}
	}
}
