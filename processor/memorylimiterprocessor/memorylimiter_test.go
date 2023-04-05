// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memorylimiterprocessor

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/iruntime"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNew(t *testing.T) {
	type args struct {
		nextConsumer        consumer.Traces
		checkInterval       time.Duration
		memoryLimitMiB      uint32
		memorySpikeLimitMiB uint32
	}
	sink := new(consumertest.TracesSink)
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "zero_checkInterval",
			args: args{
				nextConsumer: sink,
			},
			wantErr: errCheckIntervalOutOfRange,
		},
		{
			name: "zero_memAllocLimit",
			args: args{
				nextConsumer:  sink,
				checkInterval: 100 * time.Millisecond,
			},
			wantErr: errLimitOutOfRange,
		},
		{
			name: "memSpikeLimit_gt_memAllocLimit",
			args: args{
				nextConsumer:        sink,
				checkInterval:       100 * time.Millisecond,
				memoryLimitMiB:      1,
				memorySpikeLimitMiB: 2,
			},
			wantErr: errMemSpikeLimitOutOfRange,
		},
		{
			name: "success",
			args: args{
				nextConsumer:   sink,
				checkInterval:  100 * time.Millisecond,
				memoryLimitMiB: 1024,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			cfg.CheckInterval = tt.args.checkInterval
			cfg.MemoryLimitMiB = tt.args.memoryLimitMiB
			cfg.MemorySpikeLimitMiB = tt.args.memorySpikeLimitMiB
			got, err := newMemoryLimiter(processortest.NewNopCreateSettings(), cfg)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, got.start(context.Background(), componenttest.NewNopHost()))
			assert.NoError(t, got.shutdown(context.Background()))
		})
	}
}

// TestMetricsMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMetricsMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		mustRefuse: &atomic.Bool{},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: newObsReport(t),
		logger: zap.NewNop(),
	}
	mp, err := processorhelper.NewMetricsProcessor(
		context.Background(),
		processortest.NewNopCreateSettings(),
		&Config{},
		consumertest.NewNop(),
		ml.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	md := pmetric.NewMetrics()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, mp.ConsumeMetrics(ctx, md))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, mp.ConsumeMetrics(ctx, md))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.usageChecker.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.checkMemLimits()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, mp.ConsumeMetrics(ctx, md))

}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		mustRefuse: &atomic.Bool{},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: newObsReport(t),
		logger: zap.NewNop(),
	}
	tp, err := processorhelper.NewTracesProcessor(
		context.Background(),
		processortest.NewNopCreateSettings(),
		&Config{},
		consumertest.NewNop(),
		ml.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	td := ptrace.NewTraces()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, tp.ConsumeTraces(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, tp.ConsumeTraces(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.usageChecker.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.checkMemLimits()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, tp.ConsumeTraces(ctx, td))

}

// TestLogMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestLogMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		mustRefuse: &atomic.Bool{},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: newObsReport(t),
		logger: zap.NewNop(),
	}
	lp, err := processorhelper.NewLogsProcessor(
		context.Background(),
		processortest.NewNopCreateSettings(),
		&Config{},
		consumertest.NewNop(),
		ml.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	ld := plog.NewLogs()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, lp.ConsumeLogs(ctx, ld))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, lp.ConsumeLogs(ctx, ld))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.usageChecker.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.checkMemLimits()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.checkMemLimits()
	assert.Equal(t, errDataRefused, lp.ConsumeLogs(ctx, ld))
}

func TestGetDecision(t *testing.T) {
	t.Run("fixed_limit", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitMiB: 100, MemorySpikeLimitMiB: 20}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &memUsageChecker{
			memAllocLimit: 100 * mibBytes,
			memSpikeLimit: 20 * mibBytes,
		}, d)
	})
	t.Run("fixed_limit_error", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitMiB: 20, MemorySpikeLimitMiB: 100}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
	})

	t.Cleanup(func() {
		getMemoryFn = iruntime.TotalMemory
	})
	getMemoryFn = func() (uint64, error) {
		return 100 * mibBytes, nil
	}
	t.Run("percentage_limit", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitPercentage: 50, MemorySpikePercentage: 10}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &memUsageChecker{
			memAllocLimit: 50 * mibBytes,
			memSpikeLimit: 10 * mibBytes,
		}, d)
	})
	t.Run("percentage_limit_error", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitPercentage: 101, MemorySpikePercentage: 10}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
		d, err = getMemUsageChecker(&Config{MemoryLimitPercentage: 99, MemorySpikePercentage: 101}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
	})
}

func TestRefuseDecision(t *testing.T) {
	decison1000Limit30Spike30, err := newPercentageMemUsageChecker(1000, 60, 30)
	require.NoError(t, err)
	decison1000Limit60Spike50, err := newPercentageMemUsageChecker(1000, 60, 50)
	require.NoError(t, err)
	decison1000Limit40Spike20, err := newPercentageMemUsageChecker(1000, 40, 20)
	require.NoError(t, err)
	decison1000Limit40Spike60, err := newPercentageMemUsageChecker(1000, 40, 60)
	require.Error(t, err)
	assert.Nil(t, decison1000Limit40Spike60)

	tests := []struct {
		name         string
		usageChecker memUsageChecker
		ms           *runtime.MemStats
		shouldRefuse bool
	}{
		{
			name:         "should refuse over limit",
			usageChecker: *decison1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 600},
			shouldRefuse: true,
		},
		{
			name:         "should not refuse",
			usageChecker: *decison1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 100},
			shouldRefuse: false,
		},
		{
			name: "should not refuse spike, fixed usageChecker",
			usageChecker: memUsageChecker{
				memAllocLimit: 600,
				memSpikeLimit: 500,
			},
			ms:           &runtime.MemStats{Alloc: 300},
			shouldRefuse: true,
		},
		{
			name:         "should refuse, spike, percentage usageChecker",
			usageChecker: *decison1000Limit60Spike50,
			ms:           &runtime.MemStats{Alloc: 300},
			shouldRefuse: true,
		},
		{
			name:         "should refuse, spike, percentage usageChecker",
			usageChecker: *decison1000Limit40Spike20,
			ms:           &runtime.MemStats{Alloc: 250},
			shouldRefuse: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shouldRefuse := test.usageChecker.aboveSoftLimit(test.ms)
			assert.Equal(t, test.shouldRefuse, shouldRefuse)
		})
	}
}

func TestBallastSize(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.CheckInterval = 10 * time.Second
	cfg.MemoryLimitMiB = 1024
	got, err := newMemoryLimiter(processortest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)
	require.NoError(t, got.start(context.Background(), &host{ballastSize: 113}))
	assert.Equal(t, uint64(113), got.ballastSize)
	require.NoError(t, got.shutdown(context.Background()))
}

type host struct {
	ballastSize uint64
	component.Host
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	ret := make(map[component.ID]component.Component)
	ret[component.NewID("ballast")] = &ballastExtension{ballastSize: h.ballastSize}
	return ret
}

type ballastExtension struct {
	ballastSize uint64
	component.StartFunc
	component.ShutdownFunc
}

func (be *ballastExtension) GetBallastSize() uint64 {
	return be.ballastSize
}

func newObsReport(t *testing.T) *obsreport.Processor {
	set := obsreport.ProcessorSettings{
		ProcessorID:             component.NewID(typeStr),
		ProcessorCreateSettings: processortest.NewNopCreateSettings(),
	}
	set.ProcessorCreateSettings.MetricsLevel = configtelemetry.LevelNone

	proc, err := obsreport.NewProcessor(set)
	require.NoError(t, err)

	return proc
}

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

	set := processortest.NewNopCreateSettings()

	limiter, err := newMemoryLimiter(set, cfg)
	require.NoError(t, err)

	processor, err := processorhelper.NewLogsProcessor(context.Background(), processor.CreateSettings{}, cfg, exporter,
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
		return receiver.ProduceCount == exporter.DeliveredLogCount()
	}, 5*time.Second, 1*time.Millisecond)

	// Double check that the number of logs accepted by exporter matches the number of produced by receiver.
	assert.Equal(t, receiver.ProduceCount, exporter.AcceptedLogCount())

	err = processor.Shutdown(context.Background())
	require.NoError(t, err)
}
