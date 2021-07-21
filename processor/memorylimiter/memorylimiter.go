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
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/internal/iruntime"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	mibBytes = 1024 * 1024
)

var (
	// errForcedDrop will be returned to callers of ConsumeTraceData to indicate
	// that data is being dropped due to high memory usage.
	errForcedDrop = errors.New("data dropped due to high memory usage")

	// Construction errors

	errCheckIntervalOutOfRange = errors.New(
		"checkInterval must be greater than zero")

	errLimitOutOfRange = errors.New(
		"memAllocLimit or memoryLimitPercentage must be greater than zero")

	errMemSpikeLimitOutOfRange = errors.New(
		"memSpikeLimit must be smaller than memAllocLimit")

	errPercentageLimitOutOfRange = errors.New(
		"memoryLimitPercentage and memorySpikePercentage must be greater than zero and less than or equal to hundred",
	)
)

// make it overridable by tests
var getMemoryFn = iruntime.TotalMemory

type memoryLimiter struct {
	usageChecker memUsageChecker

	memCheckWait time.Duration
	ballastSize  uint64

	// forceDrop is used atomically to indicate when data should be dropped.
	forceDrop int64

	ticker *time.Ticker

	lastGCDone time.Time

	// The function to read the mem values is set as a reference to help with
	// testing different values.
	readMemStatsFn func(m *runtime.MemStats)

	// Fields used for logging.
	logger                 *zap.Logger
	configMismatchedLogged bool

	obsrep *obsreport.Processor
}

// Minimum interval between forced GC when in soft limited mode. We don't want to
// do GCs too frequently since it is a CPU-heavy operation.
const minGCIntervalWhenSoftLimited = 10 * time.Second

// newMemoryLimiter returns a new memorylimiter processor.
func newMemoryLimiter(logger *zap.Logger, cfg *Config) (*memoryLimiter, error) {
	if cfg.CheckInterval <= 0 {
		return nil, errCheckIntervalOutOfRange
	}
	if cfg.MemoryLimitMiB == 0 && cfg.MemoryLimitPercentage == 0 {
		return nil, errLimitOutOfRange
	}

	usageChecker, err := getMemUsageChecker(cfg, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Memory limiter configured",
		zap.Uint64("limit_mib", usageChecker.memAllocLimit/mibBytes),
		zap.Uint64("spike_limit_mib", usageChecker.memSpikeLimit/mibBytes),
		zap.Duration("check_interval", cfg.CheckInterval))

	ml := &memoryLimiter{
		usageChecker:   *usageChecker,
		memCheckWait:   cfg.CheckInterval,
		ticker:         time.NewTicker(cfg.CheckInterval),
		readMemStatsFn: runtime.ReadMemStats,
		logger:         logger,
		obsrep: obsreport.NewProcessor(obsreport.ProcessorSettings{
			Level:       configtelemetry.GetMetricsLevelFlagValue(),
			ProcessorID: cfg.ID(),
		}),
	}

	return ml, nil
}

func getMemUsageChecker(cfg *Config, logger *zap.Logger) (*memUsageChecker, error) {
	memAllocLimit := uint64(cfg.MemoryLimitMiB) * mibBytes
	memSpikeLimit := uint64(cfg.MemorySpikeLimitMiB) * mibBytes
	if cfg.MemoryLimitMiB != 0 {
		return newFixedMemUsageChecker(memAllocLimit, memSpikeLimit)
	}
	totalMemory, err := getMemoryFn()
	if err != nil {
		return nil, fmt.Errorf("failed to get total memory, use fixed memory settings (limit_mib): %w", err)
	}
	logger.Info("Using percentage memory limiter",
		zap.Uint64("total_memory_mib", totalMemory/mibBytes),
		zap.Uint32("limit_percentage", cfg.MemoryLimitPercentage),
		zap.Uint32("spike_limit_percentage", cfg.MemorySpikePercentage))
	return newPercentageMemUsageChecker(totalMemory, uint64(cfg.MemoryLimitPercentage), uint64(cfg.MemorySpikePercentage))
}

func (ml *memoryLimiter) start(_ context.Context, host component.Host) error {
	extensions := host.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(*ballastextension.MemoryBallast); ok {
			ml.ballastSize = ext.GetBallastSize()
			break
		}
	}

	ml.startMonitoring()
	return nil
}

func (ml *memoryLimiter) shutdown(context.Context) error {
	ml.ticker.Stop()
	return nil
}

func (ml *memoryLimiter) processTraces(ctx context.Context, td pdata.Traces) (pdata.Traces, error) {
	numSpans := td.SpanCount()
	if ml.forcingDrop() {
		// TODO: actually to be 100% sure that this is "refused" and not "dropped"
		// 	it is necessary to check the pipeline to see if this is directly connected
		// 	to a receiver (ie.: a receiver is on the call stack). For now it
		// 	assumes that the pipeline is properly configured and a receiver is on the
		// 	callstack.
		ml.obsrep.TracesRefused(ctx, numSpans)

		return td, errForcedDrop
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	ml.obsrep.TracesAccepted(ctx, numSpans)
	return td, nil
}

func (ml *memoryLimiter) processMetrics(ctx context.Context, md pdata.Metrics) (pdata.Metrics, error) {
	numDataPoints := md.DataPointCount()
	if ml.forcingDrop() {
		// TODO: actually to be 100% sure that this is "refused" and not "dropped"
		// 	it is necessary to check the pipeline to see if this is directly connected
		// 	to a receiver (ie.: a receiver is on the call stack). For now it
		// 	assumes that the pipeline is properly configured and a receiver is on the
		// 	callstack.
		ml.obsrep.MetricsRefused(ctx, numDataPoints)
		return md, errForcedDrop
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	ml.obsrep.MetricsAccepted(ctx, numDataPoints)
	return md, nil
}

func (ml *memoryLimiter) processLogs(ctx context.Context, ld pdata.Logs) (pdata.Logs, error) {
	numRecords := ld.LogRecordCount()
	if ml.forcingDrop() {
		// TODO: actually to be 100% sure that this is "refused" and not "dropped"
		// 	it is necessary to check the pipeline to see if this is directly connected
		// 	to a receiver (ie.: a receiver is on the call stack). For now it
		// 	assumes that the pipeline is properly configured and a receiver is on the
		// 	callstack.
		ml.obsrep.LogsRefused(ctx, numRecords)

		return ld, errForcedDrop
	}

	// Even if the next consumer returns error record the data as accepted by
	// this processor.
	ml.obsrep.LogsAccepted(ctx, numRecords)
	return ld, nil
}

func (ml *memoryLimiter) readMemStats() *runtime.MemStats {
	ms := &runtime.MemStats{}
	ml.readMemStatsFn(ms)
	// If proper configured ms.Alloc should be at least ml.ballastSize but since
	// a misconfiguration is possible check for that here.
	if ms.Alloc >= ml.ballastSize {
		ms.Alloc -= ml.ballastSize
	} else if !ml.configMismatchedLogged {
		// This indicates misconfiguration. Log it once.
		ml.configMismatchedLogged = true
		ml.logger.Warn(ballastSizeMibKey + " in ballast extension is likely incorrectly configured.")
	}

	return ms
}

// startMonitoring starts a ticker'd goroutine that will check memory usage
// every checkInterval period.
func (ml *memoryLimiter) startMonitoring() {
	go func() {
		for range ml.ticker.C {
			ml.checkMemLimits()
		}
	}()
}

// forcingDrop indicates when memory resources need to be released.
func (ml *memoryLimiter) forcingDrop() bool {
	return atomic.LoadInt64(&ml.forceDrop) != 0
}

func (ml *memoryLimiter) setForcingDrop(b bool) {
	var i int64
	if b {
		i = 1
	}
	atomic.StoreInt64(&ml.forceDrop, i)
}

func memstatToZapField(ms *runtime.MemStats) zap.Field {
	return zap.Uint64("cur_mem_mib", ms.Alloc/mibBytes)
}

func (ml *memoryLimiter) doGCandReadMemStats() *runtime.MemStats {
	runtime.GC()
	ml.lastGCDone = time.Now()
	ms := ml.readMemStats()
	ml.logger.Info("Memory usage after GC.", memstatToZapField(ms))
	return ms
}

func (ml *memoryLimiter) checkMemLimits() {
	ms := ml.readMemStats()

	ml.logger.Debug("Currently used memory.", memstatToZapField(ms))

	if ml.usageChecker.aboveHardLimit(ms) {
		ml.logger.Warn("Memory usage is above hard limit. Forcing a GC.", memstatToZapField(ms))
		ms = ml.doGCandReadMemStats()
	}

	// Remember current dropping state.
	wasForcingDrop := ml.forcingDrop()

	// Check if the memory usage is above the soft limit.
	mustForceDrop := ml.usageChecker.aboveSoftLimit(ms)

	if wasForcingDrop && !mustForceDrop {
		// Was previously dropping but enough memory is available now, no need to limit.
		ml.logger.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
	}

	if !wasForcingDrop && mustForceDrop {
		// We are above soft limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > minGCIntervalWhenSoftLimited {
			ml.logger.Info("Memory usage is above soft limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			mustForceDrop = ml.usageChecker.aboveSoftLimit(ms)
		}

		if mustForceDrop {
			ml.logger.Warn("Memory usage is above soft limit. Dropping data.", memstatToZapField(ms))
		}
	}

	ml.setForcingDrop(mustForceDrop)
}

type memUsageChecker struct {
	memAllocLimit uint64
	memSpikeLimit uint64
}

func (d memUsageChecker) aboveSoftLimit(ms *runtime.MemStats) bool {
	return ms.Alloc >= d.memAllocLimit-d.memSpikeLimit
}

func (d memUsageChecker) aboveHardLimit(ms *runtime.MemStats) bool {
	return ms.Alloc >= d.memAllocLimit
}

func newFixedMemUsageChecker(memAllocLimit, memSpikeLimit uint64) (*memUsageChecker, error) {
	if memSpikeLimit >= memAllocLimit {
		return nil, errMemSpikeLimitOutOfRange
	}
	if memSpikeLimit == 0 {
		// If spike limit is unspecified use 20% of mem limit.
		memSpikeLimit = memAllocLimit / 5
	}
	return &memUsageChecker{
		memAllocLimit: memAllocLimit,
		memSpikeLimit: memSpikeLimit,
	}, nil
}

func newPercentageMemUsageChecker(totalMemory uint64, percentageLimit, percentageSpike uint64) (*memUsageChecker, error) {
	if percentageLimit > 100 || percentageLimit <= 0 || percentageSpike > 100 || percentageSpike <= 0 {
		return nil, errPercentageLimitOutOfRange
	}
	return newFixedMemUsageChecker(percentageLimit*totalMemory/100, percentageSpike*totalMemory/100)
}
