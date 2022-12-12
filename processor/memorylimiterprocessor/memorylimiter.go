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

package memorylimiterprocessor // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor"

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tilinna/clock"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
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

type memorySettings struct {
	total memory.TotalFunc
	usage memory.StatsFunc
	gc    memory.GCFunc
}
type memoryLimiter struct {
	interval    time.Duration
	memObserver *observer

	logger *zap.Logger
	obsrep *obsreport.Processor

	sem  chan struct{}
	done context.CancelFunc
	wg   sync.WaitGroup
}

// newMemoryLimiter returns a new memorylimiter processor.
func newMemoryLimiter(ctx context.Context, set processor.CreateSettings, cfg *Config, opts ...func(*memorySettings)) (*memoryLimiter, error) {
	memSettings := &memorySettings{}
	for _, opt := range opts {
		opt(memSettings)
	}
	logger := set.Logger
	limiter, err := memSettings.total.NewMemChecker(
		uint64(cfg.MemoryLimitMiB),
		uint64(cfg.MemorySpikeLimitMiB),
		uint64(cfg.MemoryLimitPercentage),
		uint64(cfg.MemorySpikePercentage),
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Memory limiter configured",
		zap.Uint64("limit_mib", limiter.HardLimitMiB()),
		zap.Uint64("spike_limit_mib", limiter.SoftLimitMiB()),
		zap.Duration("check_interval", cfg.CheckInterval))

	obsrep, err := obsreport.NewProcessor(obsreport.ProcessorSettings{
		ProcessorID:             set.ID,
		ProcessorCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &memoryLimiter{
		interval: cfg.CheckInterval,
		logger:   logger,
		obsrep:   obsrep,
		memObserver: newObserver(
			ctx,
			*limiter,
			logger,
			withMemoryStats(memSettings.usage),
			withMemoryGC(memSettings.gc),
		),
		sem: make(chan struct{}, 1),
	}, nil
}

func (ml *memoryLimiter) start(ctx context.Context, host component.Host) error {
	select {
	case ml.sem <- struct{}{}:
		// acquired Sem, start processor
	default:
		return nil
	}
	if err := ml.memObserver.start(ctx, host); err != nil {
		return err
	}
	var (
		schedulerCtx context.Context
		ticker       = clock.FromContext(ctx).NewTicker(ml.interval)
	)

	schedulerCtx, ml.done = context.WithCancel(context.Background())
	ml.wg.Add(1)
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				ml.wg.Done()
				return
			case <-ticker.C:
				ml.memObserver.checkMemoryUsage()
			}
		}
	}(schedulerCtx)
	return nil
}

func (ml *memoryLimiter) shutdown(context.Context) error {
	select {
	case <-ml.sem:
		// acquired sem, shutdown processor
	default:
		return nil
	}
	// Ensure the background process finishes before shutdown returns.
	ml.done()
	ml.wg.Wait()
	return nil
}

func (ml *memoryLimiter) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	numSpans := td.SpanCount()
	if ml.memObserver.memoryExceeded() {
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

func (ml *memoryLimiter) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	numDataPoints := md.DataPointCount()
	if ml.memObserver.memoryExceeded() {
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

func (ml *memoryLimiter) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	numRecords := ld.LogRecordCount()
	if ml.memObserver.memoryExceeded() {
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
