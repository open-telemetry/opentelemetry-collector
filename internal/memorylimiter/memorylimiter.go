// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter // import "go.opentelemetry.io/collector/internal/memorylimiter"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
)

const (
	mibBytes = 1024 * 1024

	// gcEffectivenessThreshold is the minimum percentage of memory that must be
	// reclaimed by a forced GC for it to be considered effective. If GC reclaims
	// less than this percentage, the GC interval is backed off to avoid burning
	// CPU on fruitless GC cycles.
	gcEffectivenessThreshold = 5

	// maxGCBackoffInterval is the maximum interval between forced GC calls when
	// GC is repeatedly ineffective. This caps the exponential backoff to ensure
	// the system still periodically attempts GC for recovery detection.
	maxGCBackoffInterval = 2 * time.Minute
)

var (
	// ErrDataRefused will be returned to callers of ConsumeTraceData to indicate
	// that data is being refused due to high memory usage.
	ErrDataRefused = errors.New("data refused due to high memory usage")

	// GetMemoryFn and ReadMemStatsFn make it overridable by tests
	GetMemoryFn    = iruntime.TotalMemory
	ReadMemStatsFn = runtime.ReadMemStats
)

// MemoryLimiter is used to prevent out of memory situations on the collector.
type MemoryLimiter struct {
	usageChecker memUsageChecker

	memCheckWait time.Duration

	// mustRefuse is used to indicate when data should be refused.
	mustRefuse *atomic.Bool

	ticker *time.Ticker

	minGCIntervalWhenSoftLimited time.Duration
	minGCIntervalWhenHardLimited time.Duration
	lastGCDone                   time.Time

	// The functions to read the mem values and run GC are set as a reference to help with
	// testing different values.
	readMemStatsFn func(m *runtime.MemStats)
	runGCFn        func()

	// consecutiveIneffectiveGCs tracks how many consecutive forced GCs failed to
	// reclaim a meaningful amount of memory. When GC is ineffective (e.g., memory
	// is held by live references in exporter queues), the GC interval is backed off
	// exponentially to prevent a CPU-burning GC loop that starves the collector
	// and prevents recovery. See https://github.com/open-telemetry/opentelemetry-collector/issues/4981
	consecutiveIneffectiveGCs int

	// Fields used for logging.
	logger *zap.Logger

	refCounterLock sync.Mutex
	refCounter     int
	waitGroup      sync.WaitGroup
	closed         chan struct{}
}

// NewMemoryLimiter returns a new memory limiter component
func NewMemoryLimiter(cfg *Config, logger *zap.Logger) (*MemoryLimiter, error) {
	usageChecker, err := getMemUsageChecker(cfg, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Memory limiter configured",
		zap.Uint64("limit_mib", usageChecker.memAllocLimit/mibBytes),
		zap.Uint64("spike_limit_mib", usageChecker.memSpikeLimit/mibBytes),
		zap.Duration("check_interval", cfg.CheckInterval))

	return &MemoryLimiter{
		usageChecker:                 *usageChecker,
		memCheckWait:                 cfg.CheckInterval,
		ticker:                       time.NewTicker(cfg.CheckInterval),
		minGCIntervalWhenSoftLimited: cfg.MinGCIntervalWhenSoftLimited,
		minGCIntervalWhenHardLimited: cfg.MinGCIntervalWhenHardLimited,
		lastGCDone:                   time.Now(),
		readMemStatsFn:               ReadMemStatsFn,
		runGCFn:                      runtime.GC,
		logger:                       logger,
		mustRefuse:                   &atomic.Bool{},
	}, nil
}

func (ml *MemoryLimiter) Start(_ context.Context, _ component.Host) error {
	ml.refCounterLock.Lock()
	defer ml.refCounterLock.Unlock()

	ml.refCounter++
	if ml.refCounter == 1 {
		ml.closed = make(chan struct{})
		ml.waitGroup.Go(func() {
			for {
				select {
				case <-ml.ticker.C:
				case <-ml.closed:
					return
				}
				ml.CheckMemLimits()
			}
		})
	}
	return nil
}

// Shutdown resets MemoryLimiter monitoring ticker and stop monitoring
func (ml *MemoryLimiter) Shutdown(context.Context) error {
	ml.refCounterLock.Lock()
	defer ml.refCounterLock.Unlock()

	switch ml.refCounter {
	case 0:
		return nil
	case 1:
		ml.ticker.Stop()
		close(ml.closed)
		ml.waitGroup.Wait()
	}
	ml.refCounter--
	return nil
}

// MustRefuse returns true if memory has reached its configured limits
func (ml *MemoryLimiter) MustRefuse() bool {
	return ml.mustRefuse.Load()
}

func getMemUsageChecker(cfg *Config, logger *zap.Logger) (*memUsageChecker, error) {
	memAllocLimit := uint64(cfg.MemoryLimitMiB) * mibBytes
	memSpikeLimit := uint64(cfg.MemorySpikeLimitMiB) * mibBytes
	if cfg.MemoryLimitMiB != 0 {
		return newFixedMemUsageChecker(memAllocLimit, memSpikeLimit), nil
	}
	totalMemory, err := GetMemoryFn()
	if err != nil {
		return nil, fmt.Errorf("failed to get total memory, use fixed memory settings (limit_mib): %w", err)
	}
	logger.Info("Using percentage memory limiter",
		zap.Uint64("total_memory_mib", totalMemory/mibBytes),
		zap.Uint32("limit_percentage", cfg.MemoryLimitPercentage),
		zap.Uint32("spike_limit_percentage", cfg.MemorySpikePercentage))
	return newPercentageMemUsageChecker(totalMemory, uint64(cfg.MemoryLimitPercentage),
		uint64(cfg.MemorySpikePercentage)), nil
}

func (ml *MemoryLimiter) readMemStats() *runtime.MemStats {
	ms := &runtime.MemStats{}
	ml.readMemStatsFn(ms)
	return ms
}

func memstatToZapField(ms *runtime.MemStats) zap.Field {
	return zap.Uint64("cur_mem_mib", ms.Alloc/mibBytes)
}

func (ml *MemoryLimiter) doGCandReadMemStats(memAllocBeforeGC uint64) *runtime.MemStats {
	ml.runGCFn()
	ml.lastGCDone = time.Now()
	ms := ml.readMemStats()
	ml.logger.Info("Memory usage after GC.", memstatToZapField(ms))

	// Track GC effectiveness. If GC reclaimed less than gcEffectivenessThreshold%
	// of the pre-GC allocation, it is considered ineffective. This happens when
	// memory is held by live references (e.g., exporter queues, retry goroutines)
	// and cannot be freed by GC. In this case, continued forced GC only wastes CPU
	// and can starve the collector, preventing recovery.
	if memAllocBeforeGC > 0 && ms.Alloc > memAllocBeforeGC*(100-gcEffectivenessThreshold)/100 {
		ml.consecutiveIneffectiveGCs++
		if ml.consecutiveIneffectiveGCs == 1 {
			ml.logger.Warn("Forced GC did not reclaim enough memory. Will back off GC frequency to preserve CPU for recovery.",
				zap.Uint64("mem_before_gc_mib", memAllocBeforeGC/mibBytes),
				memstatToZapField(ms))
		}
	} else {
		ml.consecutiveIneffectiveGCs = 0
	}

	return ms
}

// effectiveGCInterval returns the GC interval to use, applying exponential
// backoff when consecutive GCs have been ineffective. This prevents a
// CPU-burning GC loop when memory cannot be reclaimed (e.g., held by exporter
// queues during downstream outages).
func (ml *MemoryLimiter) effectiveGCInterval(baseInterval time.Duration) time.Duration {
	if ml.consecutiveIneffectiveGCs == 0 {
		return baseInterval
	}
	// Use CheckInterval as the minimum base when the configured interval is 0.
	interval := baseInterval
	if interval == 0 {
		interval = ml.memCheckWait
	}
	// Exponential backoff: double the interval for each consecutive ineffective GC.
	for range ml.consecutiveIneffectiveGCs {
		interval *= 2
		if interval >= maxGCBackoffInterval {
			return maxGCBackoffInterval
		}
	}
	return interval
}

// CheckMemLimits inspects current memory usage against threshold and toggles mustRefuse when threshold is exceeded
func (ml *MemoryLimiter) CheckMemLimits() {
	ms := ml.readMemStats()

	ml.logger.Debug("Currently used memory.", memstatToZapField(ms))

	// Check if we are below the soft limit.
	aboveSoftLimit := ml.usageChecker.aboveSoftLimit(ms)
	if !aboveSoftLimit {
		if ml.mustRefuse.Load() {
			// Was previously refusing but enough memory is available now, no need to limit.
			ml.logger.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
		}
		ml.mustRefuse.Store(aboveSoftLimit)
		// Reset GC backoff on recovery so the next pressure event starts fresh.
		ml.consecutiveIneffectiveGCs = 0
		return
	}

	if ml.usageChecker.aboveHardLimit(ms) {
		// We are above hard limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > ml.effectiveGCInterval(ml.minGCIntervalWhenHardLimited) {
			ml.logger.Warn("Memory usage is above hard limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats(ms.Alloc)
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.usageChecker.aboveSoftLimit(ms)
		}
	} else {
		// We are above soft limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > ml.effectiveGCInterval(ml.minGCIntervalWhenSoftLimited) {
			ml.logger.Info("Memory usage is above soft limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats(ms.Alloc)
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.usageChecker.aboveSoftLimit(ms)
		}
	}

	if !ml.mustRefuse.Load() && aboveSoftLimit {
		ml.logger.Warn("Memory usage is above soft limit. Refusing data.", memstatToZapField(ms))
	}

	ml.mustRefuse.Store(aboveSoftLimit)
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

func newFixedMemUsageChecker(memAllocLimit, memSpikeLimit uint64) *memUsageChecker {
	if memSpikeLimit == 0 {
		// If spike limit is unspecified use 20% of mem limit.
		memSpikeLimit = memAllocLimit / 5
	}
	return &memUsageChecker{
		memAllocLimit: memAllocLimit,
		memSpikeLimit: memSpikeLimit,
	}
}

func newPercentageMemUsageChecker(totalMemory, percentageLimit, percentageSpike uint64) *memUsageChecker {
	return newFixedMemUsageChecker(percentageLimit*totalMemory/100, percentageSpike*totalMemory/100)
}
