// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/memorylimiter"
	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/metadatatest"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNoDataLoss(t *testing.T) {
	// Create an exporter.
	exporter := internal.NewMockExporter()

	// Mark exporter's destination unavailable. The exporter will accept data and will queue it,
	// thus increasing the memory usage of the Collector.
	exporter.SetDestAvailable(false)

	// Create a memory limiter processor.

	cfg := createDefaultConfig().(*Config)

	// Check frequently to make the test quick.
	cfg.CheckInterval = time.Millisecond * 10

	// By how much we expect memory usage to increase because of queuing up of produced data.
	const expectedMemoryIncreaseMiB = 10

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// Set the limit to current usage plus expected increase. This means initially we will not be limited.
	cfg.MemoryLimitMiB = uint32(ms.Alloc/(1024*1024) + expectedMemoryIncreaseMiB)
	cfg.MemorySpikeLimitMiB = 1

	set := processortest.NewNopSettings(metadata.Type)

	limiter, err := newMemoryLimiterProcessor(set, cfg)
	require.NoError(t, err)

	processor, err := processorhelper.NewLogs(context.Background(), processor.Settings{
		ID:                component.MustNewID("nop"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, cfg, exporter,
		limiter.processLogs,
		processorhelper.WithStart(limiter.start),
		processorhelper.WithShutdown(limiter.shutdown))
	require.NoError(t, err)

	// Create a receiver.

	receiver := &internal.MockReceiver{
		ProduceCount: 1e5, // Must produce enough logs to make sure memory increases by at least expectedMemoryIncreaseMiB
		NextConsumer: processor,
	}

	err = processor.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Start producing data.
	receiver.Start()

	// The exporter was created such that its destination is not available.
	// This will result in queuing of produced data inside the exporter and memory usage
	// will increase.

	// We must eventually hit the memory limit and the receiver must see an error from memory limiter.
	require.Eventually(t, func() bool {
		// Did last ConsumeLogs call return an error?
		return receiver.LastConsumeResult() != nil
	}, 5*time.Second, 1*time.Millisecond)

	// We are now memory limited and receiver can't produce data anymore.

	// Now make the exporter's destination available.
	exporter.SetDestAvailable(true)

	// We should now see that exporter's queue is purged and memory usage goes down.

	// Eventually we must see that receiver's ConsumeLog call returns success again.
	require.Eventually(t, func() bool {
		return receiver.LastConsumeResult() == nil
	}, 5*time.Second, 1*time.Millisecond)

	// And eventually the exporter must confirm that it delivered exact number of produced logs.
	require.Eventually(t, func() bool {
		d := exporter.DeliveredLogCount()
		t.Logf("received: %d, expected: %d\n", d, receiver.ProduceCount)
		return receiver.ProduceCount == d
	}, 5*time.Second, 100*time.Millisecond)

	// Double check that the number of logs accepted by exporter matches the number of produced by receiver.
	assert.Equal(t, receiver.ProduceCount, exporter.AcceptedLogCount())

	err = processor.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestMetricsMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMetricsMemoryPressureResponse(t *testing.T) {
	md := pmetric.NewMetrics()

	tests := []struct {
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memorylimiter.GetMemoryFn = totalMemory
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}

			ml, err := newMemoryLimiterProcessor(processortest.NewNopSettings(metadata.Type), tt.mlCfg)
			require.NoError(t, err)
			mp, err := processorhelper.NewMetrics(
				ctx,
				processortest.NewNopSettings(metadata.Type),
				tt.mlCfg,
				consumertest.NewNop(),
				ml.processMetrics,
				processorhelper.WithCapabilities(processorCapabilities),
				processorhelper.WithStart(ml.start),
				processorhelper.WithShutdown(ml.shutdown))
			require.NoError(t, err)

			assert.NoError(t, mp.Start(ctx, &host{}))
			ml.memlimiter.CheckMemLimits()
			err = mp.ConsumeMetrics(ctx, md)
			if tt.expectError {
				assert.Equal(t, memorylimiter.ErrDataRefused, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mp.Shutdown(ctx))
		})
	}
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})
}

func TestMetricsTelemetry(t *testing.T) {
	tel := componenttest.NewTelemetry()
	cfg := &Config{
		CheckInterval:         time.Second,
		MemoryLimitPercentage: 50,
		MemorySpikePercentage: 10,
	}
	metrics, err := NewFactory().CreateMetrics(context.Background(), metadatatest.NewSettings(tel), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NoError(t, metrics.Start(context.Background(), componenttest.NewNopHost()))

	md := pmetric.NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
	for range 10 {
		require.NoError(t, metrics.ConsumeMetrics(context.Background(), md))
	}
	require.NoError(t, metrics.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorAcceptedMetricPoints(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      10,
				Attributes: attribute.NewSet(attribute.String("processor", "memory_limiter")),
			},
		}, metricdatatest.IgnoreTimestamp())

	require.NoError(t, tel.Shutdown(context.Background()))
}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	td := ptrace.NewTraces()

	tests := []struct {
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memorylimiter.GetMemoryFn = totalMemory
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}

			ml, err := newMemoryLimiterProcessor(processortest.NewNopSettings(metadata.Type), tt.mlCfg)
			require.NoError(t, err)
			tp, err := processorhelper.NewTraces(
				ctx,
				processortest.NewNopSettings(metadata.Type),
				tt.mlCfg,
				consumertest.NewNop(),
				ml.processTraces,
				processorhelper.WithCapabilities(processorCapabilities),
				processorhelper.WithStart(ml.start),
				processorhelper.WithShutdown(ml.shutdown))
			require.NoError(t, err)

			assert.NoError(t, tp.Start(ctx, &host{}))
			ml.memlimiter.CheckMemLimits()
			err = tp.ConsumeTraces(ctx, td)
			if tt.expectError {
				assert.Equal(t, memorylimiter.ErrDataRefused, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, tp.Shutdown(ctx))
		})
	}
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})
}

// TestLogMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestLogMemoryPressureResponse(t *testing.T) {
	ld := plog.NewLogs()

	tests := []struct {
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memorylimiter.GetMemoryFn = totalMemory
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}

			ml, err := newMemoryLimiterProcessor(processortest.NewNopSettings(metadata.Type), tt.mlCfg)
			require.NoError(t, err)
			tp, err := processorhelper.NewLogs(
				ctx,
				processortest.NewNopSettings(metadata.Type),
				tt.mlCfg,
				consumertest.NewNop(),
				ml.processLogs,
				processorhelper.WithCapabilities(processorCapabilities),
				processorhelper.WithStart(ml.start),
				processorhelper.WithShutdown(ml.shutdown))
			require.NoError(t, err)

			assert.NoError(t, tp.Start(ctx, &host{}))
			ml.memlimiter.CheckMemLimits()
			err = tp.ConsumeLogs(ctx, ld)
			if tt.expectError {
				assert.Equal(t, memorylimiter.ErrDataRefused, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, tp.Shutdown(ctx))
		})
	}
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})
}

// TestProfileMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestProfileMemoryPressureResponse(t *testing.T) {
	pd := pprofile.NewProfiles()

	tests := []struct {
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memorylimiter.GetMemoryFn = totalMemory
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}

			ml, err := newMemoryLimiterProcessor(processortest.NewNopSettings(metadata.Type), tt.mlCfg)
			require.NoError(t, err)
			tp, err := xprocessorhelper.NewProfiles(
				ctx,
				processortest.NewNopSettings(metadata.Type),
				tt.mlCfg,
				consumertest.NewNop(),
				ml.processProfiles,
				xprocessorhelper.WithCapabilities(processorCapabilities),
				xprocessorhelper.WithStart(ml.start),
				xprocessorhelper.WithShutdown(ml.shutdown))
			require.NoError(t, err)

			assert.NoError(t, tp.Start(ctx, &host{}))
			ml.memlimiter.CheckMemLimits()
			err = tp.ConsumeProfiles(ctx, pd)
			if tt.expectError {
				assert.Equal(t, memorylimiter.ErrDataRefused, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, tp.Shutdown(ctx))
		})
	}
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})
}

type host struct {
	component.Host
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	ret := make(map[component.ID]component.Component)
	return ret
}

func totalMemory() (uint64, error) {
	return uint64(2048), nil
}
