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

package memorylimiter

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/internal/iruntime"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor/processorhelper"
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
			got, err := newMemoryLimiter(zap.NewNop(), cfg)
			if err != tt.wantErr {
				t.Errorf("newMemoryLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				assert.NoError(t, got.shutdown(context.Background()))
			}
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
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: obsreport.NewProcessor(obsreport.ProcessorSettings{
			Level:       configtelemetry.LevelNone,
			ProcessorID: config.NewID(typeStr),
		}),

		logger: zap.NewNop(),
	}
	mp, err := processorhelper.NewMetricsProcessor(
		&Config{
			ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		},
		consumertest.NewNop(),
		ml.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	md := pdata.NewMetrics()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

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
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: obsreport.NewProcessor(obsreport.ProcessorSettings{
			Level:       configtelemetry.LevelNone,
			ProcessorID: config.NewID(typeStr),
		}),
		logger: zap.NewNop(),
	}
	tp, err := processorhelper.NewTracesProcessor(
		&Config{
			ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		},
		consumertest.NewNop(),
		ml.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	td := pdata.NewTraces()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

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
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

}

// TestLogMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestLogMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		obsrep: obsreport.NewProcessor(obsreport.ProcessorSettings{
			Level:       configtelemetry.LevelNone,
			ProcessorID: config.NewID(typeStr),
		}),
		logger: zap.NewNop(),
	}
	lp, err := processorhelper.NewLogsProcessor(
		&Config{
			ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		},
		consumertest.NewNop(),
		ml.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	ld := pdata.NewLogs()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.checkMemLimits()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.checkMemLimits()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.checkMemLimits()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

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
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))
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

func TestDropDecision(t *testing.T) {
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
		shouldDrop   bool
	}{
		{
			name:         "should drop over limit",
			usageChecker: *decison1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 600},
			shouldDrop:   true,
		},
		{
			name:         "should not drop",
			usageChecker: *decison1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 100},
			shouldDrop:   false,
		},
		{
			name: "should not drop spike, fixed usageChecker",
			usageChecker: memUsageChecker{
				memAllocLimit: 600,
				memSpikeLimit: 500,
			},
			ms:         &runtime.MemStats{Alloc: 300},
			shouldDrop: true,
		},
		{
			name:         "should drop, spike, percentage usageChecker",
			usageChecker: *decison1000Limit60Spike50,
			ms:           &runtime.MemStats{Alloc: 300},
			shouldDrop:   true,
		},
		{
			name:         "should drop, spike, percentage usageChecker",
			usageChecker: *decison1000Limit40Spike20,
			ms:           &runtime.MemStats{Alloc: 250},
			shouldDrop:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shouldDrop := test.usageChecker.aboveSoftLimit(test.ms)
			assert.Equal(t, test.shouldDrop, shouldDrop)
		})
	}
}

func TestBallastSizeMiB(t *testing.T) {
	ctx := context.Background()
	ballastExtFactory := ballastextension.NewFactory()
	ballastExtCfg := ballastExtFactory.CreateDefaultConfig().(*ballastextension.Config)
	ballastExtCfg.SizeMiB = 100
	extCreateSet := componenttest.NewNopExtensionCreateSettings()

	tests := []struct {
		name                          string
		ballastExtBallastSizeSetting  uint64
		expectedMemLimiterBallastSize uint64
		expectResult                  bool
	}{
		{
			name:                          "ballast size matched",
			ballastExtBallastSizeSetting:  100,
			expectedMemLimiterBallastSize: 100,
			expectResult:                  true,
		},
		{
			name:                          "ballast size not matched",
			ballastExtBallastSizeSetting:  1000,
			expectedMemLimiterBallastSize: 100,
			expectResult:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ballastExtCfg.SizeMiB = tt.ballastExtBallastSizeSetting
			ballastExt, _ := ballastExtFactory.CreateExtension(ctx, extCreateSet, ballastExtCfg)
			ballastExt.Start(ctx, nil)
			assert.Equal(t, tt.expectResult, tt.expectedMemLimiterBallastSize*mibBytes == ballastExt.(*ballastextension.MemoryBallast).GetBallastSize())
		})
	}
}
