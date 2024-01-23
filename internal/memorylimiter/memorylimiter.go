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
	"go.opentelemetry.io/collector/internal/iruntime"
)

const (
	mibBytes = 1024 * 1024

	// Minimum interval between forced GC when in soft limited mode. We don't want to
	// do GCs too frequently since it is a CPU-heavy operation.
	minGCIntervalWhenSoftLimited = 10 * time.Second
)

var (
	// ErrDataRefused will be returned to callers of ConsumeTraceData to indicate
	// that data is being refused due to high memory usage.
	ErrDataRefused = errors.New("data refused due to high memory usage")

	// ErrShutdownNotStarted indicates no memorylimiter has not start when shutdown
	ErrShutdownNotStarted = errors.New("no existing monitoring routine is running")

	// GetMemoryFn and ReadMemStatsFn make it overridable by tests
	GetMemoryFn    = iruntime.TotalMemory
	ReadMemStatsFn = runtime.ReadMemStats
)

// MemoryLimiter is used to prevent out of memory situations on the collector.
type MemoryLimiter struct {
	usageChecker memUsageChecker

	memCheckWait time.Duration
	ballastSize  uint64

	// mustRefuse is used to indicate when data should be refused.
	mustRefuse *atomic.Bool

	ticker *time.Ticker

	lastGCDone time.Time

	// The function to read the mem values is set as a reference to help with
	// testing different values.
	readMemStatsFn func(m *runtime.MemStats)

	// Fields used for logging.
	logger                 *zap.Logger
	configMismatchedLogged bool

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
		usageChecker:   *usageChecker,
		memCheckWait:   cfg.CheckInterval,
		ticker:         time.NewTicker(cfg.CheckInterval),
		readMemStatsFn: ReadMemStatsFn,
		logger:         logger,
		mustRefuse:     &atomic.Bool{},
	}, nil
}

// startMonitoring starts a single ticker'd goroutine per instance
// that will check memory usage every checkInterval period.
func (ml *MemoryLimiter) startMonitoring() {
	ml.refCounterLock.Lock()
	defer ml.refCounterLock.Unlock()

	ml.refCounter++
	if ml.refCounter == 1 {
		ml.closed = make(chan struct{})
		ml.waitGroup.Add(1)
		go func() {
			defer ml.waitGroup.Done()

			for {
				select {
				case <-ml.ticker.C:
				case <-ml.closed:
					return
				}
				ml.CheckMemLimits()
			}
		}()
	}
}

func (ml *MemoryLimiter) Start(_ context.Context, host component.Host) error {
	extensions := host.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(interface{ GetBallastSize() uint64 }); ok {
			ml.ballastSize = ext.GetBallastSize()
			break
		}
	}
	ml.startMonitoring()
	return nil
}

// Shutdown resets MemoryLimiter monitoring ticker and stop monitoring
func (ml *MemoryLimiter) Shutdown(context.Context) error {
	ml.refCounterLock.Lock()
	defer ml.refCounterLock.Unlock()

	if ml.refCounter == 0 {
		return ErrShutdownNotStarted
	} else if ml.refCounter == 1 {
		ml.ticker.Stop()
		close(ml.closed)
		ml.waitGroup.Wait()
	}
	ml.refCounter--
	return nil
}

// MustRefuse returns if the caller should deny because memory has reached it's configured limits
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
	// If proper configured ms.Alloc should be at least ml.ballastSize but since
	// a misconfiguration is possible check for that here.
	if ms.Alloc >= ml.ballastSize {
		ms.Alloc -= ml.ballastSize
	} else if !ml.configMismatchedLogged {
		// This indicates misconfiguration. Log it once.
		ml.configMismatchedLogged = true
		ml.logger.Warn(`"size_mib" in ballast extension is likely incorrectly configured.`)
	}

	return ms
}

func memstatToZapField(ms *runtime.MemStats) zap.Field {
	return zap.Uint64("cur_mem_mib", ms.Alloc/mibBytes)
}

func (ml *MemoryLimiter) doGCandReadMemStats() *runtime.MemStats {
	runtime.GC()
	ml.lastGCDone = time.Now()
	ms := ml.readMemStats()
	ml.logger.Info("Memory usage after GC.", memstatToZapField(ms))
	return ms
}

// CheckMemLimits inspects current memory usage against threshold and toggle mustRefuse when threshold is exceeded
func (ml *MemoryLimiter) CheckMemLimits() {
	ms := ml.readMemStats()

	ml.logger.Debug("Currently used memory.", memstatToZapField(ms))

	if ml.usageChecker.aboveHardLimit(ms) {
		ml.logger.Warn("Memory usage is above hard limit. Forcing a GC.", memstatToZapField(ms))
		ms = ml.doGCandReadMemStats()
	}

	// Remember current state.
	wasRefusing := ml.mustRefuse.Load()

	// Check if the memory usage is above the soft limit.
	mustRefuse := ml.usageChecker.aboveSoftLimit(ms)

	if wasRefusing && !mustRefuse {
		// Was previously refusing but enough memory is available now, no need to limit.
		ml.logger.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
	}

	if !wasRefusing && mustRefuse {
		// We are above soft limit, do a GC if it wasn't done recently and see if
		// it brings memory usage below the soft limit.
		if time.Since(ml.lastGCDone) > minGCIntervalWhenSoftLimited {
			ml.logger.Info("Memory usage is above soft limit. Forcing a GC.", memstatToZapField(ms))
			ms = ml.doGCandReadMemStats()
			// Check the limit again to see if GC helped.
			mustRefuse = ml.usageChecker.aboveSoftLimit(ms)
		}

		if mustRefuse {
			ml.logger.Warn("Memory usage is above soft limit. Refusing data.", memstatToZapField(ms))
		}
	}

	ml.mustRefuse.Store(mustRefuse)
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

func newPercentageMemUsageChecker(totalMemory uint64, percentageLimit, percentageSpike uint64) *memUsageChecker {
	return newFixedMemUsageChecker(percentageLimit*totalMemory/100, percentageSpike*totalMemory/100)
}
