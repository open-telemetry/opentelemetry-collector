// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
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
			name:    "nil_nextConsumer",
			wantErr: errNilNextConsumer,
		},
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
			cfg := generateDefaultConfig()
			cfg.CheckInterval = tt.args.checkInterval
			cfg.MemoryLimitMiB = tt.args.memoryLimitMiB
			cfg.MemorySpikeLimitMiB = tt.args.memorySpikeLimitMiB
			got, err := newMemoryLimiter(zap.NewNop(), tt.args.nextConsumer, nil, cfg)
			if err != tt.wantErr {
				t.Errorf("newMemoryLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				assert.NoError(t, got.Shutdown(context.Background()))
			}
		})
	}
}

// TestMetricsMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMetricsMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	sink := new(exportertest.SinkMetricsExporter)
	ml := &memoryLimiter{
		metricsConsumer: sink,
		memAllocLimit:   1024,
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}

	ctx := context.Background()
	td := data.NewMetricData()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetrics(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetrics(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetrics(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetrics(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetrics(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetrics(ctx, td))

}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	sink := new(exportertest.SinkTraceExporter)
	ml := &memoryLimiter{
		traceConsumer: sink,
		memAllocLimit: 1024,
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}

	ctx := context.Background()
	td := pdata.NewTraces()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraces(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraces(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraces(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraces(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraces(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraces(ctx, td))

}
