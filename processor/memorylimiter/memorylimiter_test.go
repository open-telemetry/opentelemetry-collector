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
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
)

func TestNew(t *testing.T) {
	type args struct {
		nextConsumer  consumer.TraceConsumer
		checkInterval time.Duration
		memAllocLimit uint64
		memSpikeLimit uint64
		ballastSize   uint64
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
				nextConsumer:  sink,
				checkInterval: 100 * time.Millisecond,
				memAllocLimit: 1024,
				memSpikeLimit: 2048,
			},
			wantErr: errMemSpikeLimitOutOfRange,
		},
		{
			name: "success",
			args: args{
				nextConsumer:  sink,
				checkInterval: 100 * time.Millisecond,
				memAllocLimit: 1e10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(
				"test",
				tt.args.nextConsumer,
				nil,
				tt.args.checkInterval,
				tt.args.memAllocLimit,
				tt.args.memSpikeLimit,
				tt.args.ballastSize,
				zap.NewNop())
			if err != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				assert.NoError(t, got.Shutdown())
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
	td := consumerdata.MetricsData{}

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetricsData(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetricsData(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetricsData(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetricsData(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, ml.ConsumeMetricsData(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeMetricsData(ctx, td))

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
	td := consumerdata.TraceData{}

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraceData(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraceData(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraceData(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraceData(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, ml.ConsumeTraceData(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, ml.ConsumeTraceData(ctx, td))

}
