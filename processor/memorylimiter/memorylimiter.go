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
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

var (
	// errForcedDrop will be returned to callers of ConsumeTraceData to indicate
	// that data is being dropped due to high memory usage.
	errForcedDrop = errors.New("data dropped due to high memory usage")

	// Construction errors

	errNilNextConsumer = errors.New("nil nextConsumer")

	errCheckIntervalOutOfRange = errors.New(
		"checkInterval must be greater than zero")

	errMemAllocLimitOutOfRange = errors.New(
		"memAllocLimit must be greater than zero")

	errMemSpikeLimitOutOfRange = errors.New(
		"memSpikeLimit must be smaller than memAllocLimit")
)

type memoryLimiter struct {
	traceConsumer   consumer.TraceConsumer
	metricsConsumer consumer.MetricsConsumer

	memAllocLimit uint64
	memSpikeLimit uint64
	memCheckWait  time.Duration
	ballastSize   uint64

	// forceDrop is used atomically to indicate when data should be dropped.
	forceDrop int64

	ticker *time.Ticker

	// The function to read the mem values is set as a reference to help with
	// testing different values.
	readMemStatsFn func(m *runtime.MemStats)

	statsTags []tag.Mutator

	// Fields used for logging.
	procName               string
	logger                 *zap.Logger
	configMismatchedLogged bool
}

var _ processor.DualTypeProcessor = (*memoryLimiter)(nil)

// New returns a new memorylimiter processor.
func New(
	name string,
	traceConsumer consumer.TraceConsumer,
	metricsConsumer consumer.MetricsConsumer,
	checkInterval time.Duration,
	memAllocLimit uint64,
	memSpikeLimit uint64,
	ballastSize uint64,
	logger *zap.Logger,
) (processor.DualTypeProcessor, error) {

	if traceConsumer == nil && metricsConsumer == nil {
		return nil, errNilNextConsumer
	}
	if checkInterval <= 0 {
		return nil, errCheckIntervalOutOfRange
	}
	if memAllocLimit == 0 {
		return nil, errMemAllocLimitOutOfRange
	}
	if memSpikeLimit >= memAllocLimit {
		return nil, errMemSpikeLimitOutOfRange
	}

	ml := &memoryLimiter{
		traceConsumer:   traceConsumer,
		metricsConsumer: metricsConsumer,
		memAllocLimit:   memAllocLimit,
		memSpikeLimit:   memSpikeLimit,
		memCheckWait:    checkInterval,
		ballastSize:     ballastSize,
		ticker:          time.NewTicker(checkInterval),
		readMemStatsFn:  runtime.ReadMemStats,
		statsTags:       statsTagsForBatch(name),
		procName:        name,
		logger:          logger,
	}

	initMetrics()

	ml.startMonitoring()

	return ml, nil
}

func (ml *memoryLimiter) ConsumeTraceData(
	ctx context.Context,
	td consumerdata.TraceData,
) error {

	if ml.forcingDrop() {
		numSpans := len(td.Spans)
		stats.RecordWithTags(
			context.Background(),
			ml.statsTags,
			StatDroppedSpanCount.M(int64(numSpans)))

		return errForcedDrop
	}
	return ml.traceConsumer.ConsumeTraceData(ctx, td)
}

func (ml *memoryLimiter) ConsumeMetricsData(
	ctx context.Context,
	md consumerdata.MetricsData,
) error {

	if ml.forcingDrop() {
		numMetrics := len(md.Metrics)
		stats.RecordWithTags(
			context.Background(),
			ml.statsTags,
			StatDroppedMetricCount.M(int64(numMetrics)))

		return errForcedDrop
	}
	return ml.metricsConsumer.ConsumeMetricsData(ctx, md)
}

func (ml *memoryLimiter) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

func (ml *memoryLimiter) Start(host component.Host) error {
	return nil
}

func (ml *memoryLimiter) Shutdown() error {
	ml.ticker.Stop()
	return nil
}

func (ml *memoryLimiter) readMemStats(ms *runtime.MemStats) {
	ml.readMemStatsFn(ms)
	// If proper configured ms.Alloc should be at least ml.ballastSize but since
	// a misconfiguration is possible check for that here.
	if ms.Alloc >= ml.ballastSize {
		ms.Alloc -= ml.ballastSize
	} else {
		// This indicates misconfiguration. Log it once.
		if !ml.configMismatchedLogged {
			ml.configMismatchedLogged = true
			ml.logger.Warn(typeStr+" is likely incorrectly configured. "+ballastSizeMibKey+
				" must be set equal to --mem-ballast-size-mib command line option.",
				zap.String("processor", ml.procName))
		}
	}
}

// startMonitoring starts a ticker'd goroutine that will check memory usage
// every checkInterval period.
func (ml *memoryLimiter) startMonitoring() {
	go func() {
		for range ml.ticker.C {
			ml.memCheck()
		}
	}()
}

// forcingDrop indicates when memory resources need to be released.
func (ml *memoryLimiter) forcingDrop() bool {
	return atomic.LoadInt64(&ml.forceDrop) != 0
}

func (ml *memoryLimiter) memCheck() {
	ms := &runtime.MemStats{}
	ml.readMemStats(ms)
	ml.memLimiting(ms)
}

func (ml *memoryLimiter) shouldForceDrop(ms *runtime.MemStats) bool {
	return ml.memAllocLimit <= ms.Alloc || ml.memAllocLimit-ms.Alloc <= ml.memSpikeLimit
}

func (ml *memoryLimiter) memLimiting(ms *runtime.MemStats) {
	if !ml.shouldForceDrop(ms) {
		atomic.StoreInt64(&ml.forceDrop, 0)
	} else {
		atomic.StoreInt64(&ml.forceDrop, 1)
		// Force a GC at this point and see if this is enough to get to
		// the desired level.
		runtime.GC()
	}
}
