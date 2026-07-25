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
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
)

const (
	mibBytes = 1024 * 1024

	// gcEffectivenessThreshold is the minimum percentage of memory that must be
	// reclaimed by a forced GC for it to be considered effective. If GC reclaims
	// less than this percentage, the GC interval is backed off to avoid burning
	// CPU on fruitless GC cycles.
	gcEffectivenessThreshold = 5
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
	maxGCIntervalWhenSoftLimited time.Duration
	maxGCIntervalWhenHardLimited time.Duration
	lastGCDone                   time.Time

	// The functions to read the mem values and run GC are set as a reference to help with
	// testing different values.
	readMemStatsFn func(m *runtime.MemStats)
	runGCFn        func()

	// currentSoftGCInterval and currentHardGCInterval are per-path dynamic GC
	// intervals that grow when forced GCs on that path fail to reclaim memory
	// (held by live references in exporter queues during downstream outages).
	// Each resets to zero whenever a GC on its path is effective or memory
	// drops on its own, and doubles after each ineffective GC on its path up
	// to the corresponding maxGCIntervalWhen(Hard|Soft)Limited cap.
	//
	// The floor for doubling is max(minGCIntervalWhen(Hard|Soft)Limited,
	// memCheckWait*0.95). When the configured min is below memCheckWait the
	// floor lands just under memCheckWait — small enough that the next-tick
	// gate (time.Since(lastGCDone) > floor) reliably clears, since lastGCDone
	// is stamped after the forced GC completes and the next ticker fire may
	// land slightly under a full memCheckWait afterward (the GC's wall-clock
	// cost and timer slop can eat into the interval). Using exactly
	// memCheckWait would let the gate fail on those ticks and reduce forced
	// GC to every other tick.
	//
	// State is kept per-path so that disabling backoff on one path (max=0)
	// does not influence gating on the other path.
	currentSoftGCInterval time.Duration
	currentHardGCInterval time.Duration

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

	host component.Host
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
		maxGCIntervalWhenSoftLimited: cfg.MaxGCIntervalWhenSoftLimited,
		maxGCIntervalWhenHardLimited: cfg.MaxGCIntervalWhenHardLimited,
		lastStats:                    &runtime.MemStats{},
		lastGCDone:                   time.Now(),
		readMemStatsFn:               ReadMemStatsFn,
		runGCFn:                      runtime.GC,
		logger:                       logger,
		mustRefuse:                   &atomic.Bool{},
	}, nil
}

func (ml *MemoryLimiter) Start(_ context.Context, host component.Host) error {
	ml.host = host
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

// checkLimitAndBackoff returns whether memory is above the soft limit. When
// didGC is true (called immediately after a forced GC), it also advances both
// per-path backoff intervals in lockstep so the inactive path is not caught
// flat-footed when memory pressure shifts between soft and hard. A GC is
// effective if it drops memory below the soft limit or reclaims at least
// gcEffectivenessThreshold%; an effective observation resets both intervals
// to zero because memory really did come down (a global signal).
func (ml *MemoryLimiter) checkLimitAndBackoff(ms *runtime.MemStats, didGC bool) (aboveSoftLimit bool) {
	aboveSoftLimit = ml.usageChecker.aboveSoftLimit(ms)
	gcWasEffective := !aboveSoftLimit ||
		ms.Alloc <= ml.lastStats.Alloc*(100-gcEffectivenessThreshold)/100
	ml.lastStats = ms
	if gcWasEffective {
		ml.currentSoftGCInterval = 0
		ml.currentHardGCInterval = 0
		return aboveSoftLimit
	}
	if !didGC {
		return aboveSoftLimit
	}
	ml.currentSoftGCInterval = ml.nextBackoffInterval(ml.currentSoftGCInterval, ml.minGCIntervalWhenSoftLimited, ml.maxGCIntervalWhenSoftLimited)
	ml.currentHardGCInterval = ml.nextBackoffInterval(ml.currentHardGCInterval, ml.minGCIntervalWhenHardLimited, ml.maxGCIntervalWhenHardLimited)
	ml.logger.Warn("Forced GC did not reclaim enough memory. Will back off GC frequency to preserve CPU for recovery.",
		zap.Duration("next_soft_gc_interval", ml.currentSoftGCInterval),
		zap.Duration("next_hard_gc_interval", ml.currentHardGCInterval),
		memstatToZapField(ms))
	return aboveSoftLimit
}

// nextBackoffInterval returns the new backoff interval for one path after an
// ineffective forced GC. When configMax is 0 the path's backoff is disabled
// and the interval is pinned to configMin (so the path keeps GCing every
// configMin, or every check interval if configMin is also 0). Otherwise the
// interval starts at max(configMin, 0.95*memCheckWait) — see the comment on
// currentSoftGCInterval for why 0.95 — doubles on each ineffective GC, and
// is capped at max(configMax, configMin).
func (ml *MemoryLimiter) nextBackoffInterval(current, configMin, configMax time.Duration) time.Duration {
	if configMax == 0 {
		return configMin
	}
	minInterval := max(configMin, time.Duration(float64(ml.memCheckWait)*0.95))
	maxInterval := max(configMax, configMin)
	return min(max(current, minInterval)*2, maxInterval)
}

// CheckMemLimits inspects current memory usage against threshold and toggles mustRefuse when threshold is exceeded
func (ml *MemoryLimiter) CheckMemLimits() {
	ms := ml.readMemStats()

	ml.logger.Debug("Currently used memory.", memstatToZapField(ms))

	// Top-of-tick classification. didGC=false because no forced GC has run
	// yet this tick. If memory dropped on its own (Go runtime GC, queue
	// drained), this resets both per-path currentGCIntervals so we won't
	// suppress the next forced GC unnecessarily.
	aboveSoftLimit := ml.checkLimitAndBackoff(ms, false)
	if !aboveSoftLimit {
		if ml.mustRefuse.Load() {
			// Was previously refusing but enough memory is available now, no need to limit.
			ml.logger.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
		}
		componentstatus.ReportStatus(ml.host, componentstatus.NewEvent(componentstatus.StatusOK))
		ml.mustRefuse.Store(aboveSoftLimit)
		return
	}

	if ml.usageChecker.aboveHardLimit(ms) {
		gateInterval := max(ml.minGCIntervalWhenHardLimited, ml.currentHardGCInterval)
		// We are above hard limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > gateInterval {
			ml.logger.Warn("Memory usage is above hard limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.checkLimitAndBackoff(ms, true)
		}
	} else {
		gateInterval := max(ml.minGCIntervalWhenSoftLimited, ml.currentSoftGCInterval)
		// We are above soft limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > gateInterval {
			ml.logger.Info("Memory usage is above soft limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			aboveSoftLimit = ml.checkLimitAndBackoff(ms, true)
		}
	}

	if !ml.mustRefuse.Load() && aboveSoftLimit {
		componentstatus.ReportStatus(ml.host, componentstatus.NewRecoverableErrorEvent(ErrDataRefused))
		ml.logger.Warn("Memory usage is above soft limit. Refusing data.", memstatToZapField(ms))
	}

	if !aboveSoftLimit {
		// GC brought memory back within limits in this same tick — report OK immediately.
		componentstatus.ReportStatus(ml.host, componentstatus.NewEvent(componentstatus.StatusOK))
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
