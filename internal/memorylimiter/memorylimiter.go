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
	maxGCBackoffInterval = 30 * time.Second
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

	// backoffOnIneffectiveGC enables exponential backoff between forced GCs
	// when consecutive collections fail to reclaim memory. See #4981.
	backoffOnIneffectiveGC bool

	// currentGCInterval is the dynamic GC interval that grows when forced GCs
	// fail to reclaim memory (held by live references in exporter queues during
	// downstream outages). It resets to zero whenever a GC is effective, and
	// doubles after each ineffective GC up to maxGCBackoffInterval. A floor of
	// max(minGCIntervalWhen*Limited, memCheckWait*0.95) is applied before
	// doubling so user-configured pacing is never reduced.
	currentGCInterval time.Duration

	// lastStats is the most recent runtime.MemStats observation. Comparing
	// current Alloc against lastStats.Alloc is how checkLimitAndBackoff
	// classifies a forced GC as effective when memory drops by at least 5%.
	lastStats *runtime.MemStats

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
		backoffOnIneffectiveGC:       cfg.BackoffOnIneffectiveGC,
		lastStats:                    &runtime.MemStats{},
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

func (ml *MemoryLimiter) doGCandReadMemStats() *runtime.MemStats {
	ml.runGCFn()
	ml.lastGCDone = time.Now()
	ms := ml.readMemStats()
	ml.logger.Info("Memory usage after GC.", memstatToZapField(ms))
	return ms
}

// checkLimitAndBackoff classifies a memory observation against the soft limit
// and, when backoff is enabled, updates currentGCInterval. A GC is considered
// effective if it either drops memory below the soft limit or reclaims at
// least gcEffectivenessThreshold% of the previously observed Alloc. An
// ineffective forced GC seeds currentGCInterval to a floor of
// max(configMin, memCheckWait*0.95) and then doubles it (capped at
// max(maxGCBackoffInterval, configMin)). The floor only applies after a
// confirmed ineffective forced GC, leaving happy-path GC pacing identical
// to the pre-backoff behavior.
//
// configMin is the branch-specific min interval and is only consulted when
// didGC is true. Callers passing didGC=false (top-of-tick) may pass 0.
//
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4981.
func (ml *MemoryLimiter) checkLimitAndBackoff(ms *runtime.MemStats, didGC bool, configMin time.Duration) (aboveSoftLimit bool) {
	aboveSoftLimit = ml.usageChecker.aboveSoftLimit(ms)
	if !ml.backoffOnIneffectiveGC {
		ml.lastStats = ms
		return aboveSoftLimit
	}
	gcWasEffective := !aboveSoftLimit ||
		ms.Alloc <= ml.lastStats.Alloc*(100-gcEffectivenessThreshold)/100
	ml.lastStats = ms
	if gcWasEffective {
		ml.currentGCInterval = 0
		return aboveSoftLimit
	}
	if !didGC {
		return aboveSoftLimit
	}
	floor := backoffFloor(configMin, ml.memCheckWait)
	if ml.currentGCInterval < floor {
		ml.currentGCInterval = floor
	}
	intervalCap := max(maxGCBackoffInterval, configMin)
	doubled := ml.currentGCInterval * 2
	if doubled > intervalCap || doubled < ml.currentGCInterval {
		doubled = intervalCap
	}
	ml.currentGCInterval = doubled
	ml.logger.Warn("Forced GC did not reclaim enough memory. Will back off GC frequency to preserve CPU for recovery.",
		zap.Duration("next_gc_interval", ml.currentGCInterval),
		memstatToZapField(ms))
	return aboveSoftLimit
}

// backoffFloor returns the minimum value currentGCInterval should take after
// the first ineffective GC of a streak. The user-configured min interval is
// a floor; 95% of the check tick is the de facto minimum since CheckMemLimits
// cannot be invoked more often than that. Whichever is larger wins.
func backoffFloor(configMin, checkInterval time.Duration) time.Duration {
	return max(configMin, time.Duration(float64(checkInterval)*0.95))
}

// CheckMemLimits inspects current memory usage against threshold and toggles mustRefuse when threshold is exceeded
func (ml *MemoryLimiter) CheckMemLimits() {
	ms := ml.readMemStats()

	ml.logger.Debug("Currently used memory.", memstatToZapField(ms))

	// Top-of-tick classification. didGC=false because no forced GC has run
	// yet this tick. If memory dropped on its own (Go runtime GC, queue
	// drained), this also resets currentGCInterval so we won't suppress the
	// next forced GC unnecessarily.
	aboveSoftLimit := ml.checkLimitAndBackoff(ms, false, 0)
	if !aboveSoftLimit {
		if ml.mustRefuse.Load() {
			// Was previously refusing but enough memory is available now, no need to limit.
			ml.logger.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
		}
		ml.mustRefuse.Store(aboveSoftLimit)
		return
	}

	if ml.usageChecker.aboveHardLimit(ms) {
		gateInterval := max(ml.minGCIntervalWhenHardLimited, ml.currentGCInterval)
		// We are above hard limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > gateInterval {
			ml.logger.Warn("Memory usage is above hard limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.checkLimitAndBackoff(ms, true, ml.minGCIntervalWhenHardLimited)
		}
	} else {
		gateInterval := max(ml.minGCIntervalWhenSoftLimited, ml.currentGCInterval)
		// We are above soft limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > gateInterval {
			ml.logger.Info("Memory usage is above soft limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.checkLimitAndBackoff(ms, true, ml.minGCIntervalWhenSoftLimited)
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
