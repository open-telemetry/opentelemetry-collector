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

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestNew(t *testing.T) {
	type args struct {
		nextConsumer        consumer.TraceConsumer
		checkInterval       time.Duration
		memoryLimitMiB      uint32
		memorySpikeLimitMiB uint32
	}
	sink := new(exportertest.SinkTraceExporter)
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
			wantErr: errMemAllocLimitOutOfRange,
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
		memAllocLimit: 1024,
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	mp, err := processorhelper.NewMetricsProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		exportertest.NewNopMetricsExporter(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	md := dataold.NewMetricData()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(md)))

}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		memAllocLimit: 1024,
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	tp, err := processorhelper.NewTraceProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		exportertest.NewNopTraceExporter(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	td := pdata.NewTraces()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

}

// TestLogMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestLogMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		memAllocLimit: 1024,
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	lp, err := processorhelper.NewLogsProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		exportertest.NewNopLogsExporter(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	ld := pdata.NewLogs()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))
}
