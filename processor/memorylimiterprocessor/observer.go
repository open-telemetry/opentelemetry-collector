// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memorylimiterprocessor // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor"

import (
	"context"
	"runtime"
	"time"

	"github.com/tilinna/clock"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
)

const (
	// minGCIntervalWhenSoftLimited is the min interval between forced GC when in soft limited mode.
	// We don't want to do GCs too frequently since it is a CPU-heavy operation.
	minGCIntervalWhenSoftLimited = 10 * time.Second

	mibBytes = 1024 * 1024
)

// observer is used to check the current memory usage
// ether compared against a static amount or the total
// limit as defined against the ballast.
type observer struct {
	limits      memory.Checker
	ballastSize uint64

	stats memory.StatsFunc
	gc    memory.GCFunc
	clock clock.Clock

	log *zap.Logger

	exceeded           *atomic.Bool
	reportedMismatched bool
	lastGC             time.Time
}

type option func(o *observer)

func withMemoryStats(fn memory.StatsFunc) option {
	return func(o *observer) {
		o.stats = fn
	}
}

func withMemoryGC(fn memory.GCFunc) option {
	return func(o *observer) {
		o.gc = fn
	}
}

func memstatToZapField(ms *runtime.MemStats) zap.Field {
	return zap.Uint64("cur_mem_mib", ms.Alloc/mibBytes)
}

func newObserver(ctx context.Context, limits memory.Checker, log *zap.Logger, opts ...option) *observer {
	o := &observer{
		limits:   limits,
		clock:    clock.FromContext(ctx),
		log:      log,
		exceeded: atomic.NewBool(false),
		lastGC:   clock.FromContext(ctx).Now(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *observer) start(_ context.Context, host component.Host) error { //nolint:unparam // Implements to component.StartFunc
	for _, extension := range host.GetExtensions() {
		if ext, ok := extension.(interface{ GetBallastSize() uint64 }); ok {
			o.ballastSize = ext.GetBallastSize()
			break
		}
	}
	return nil
}

// memoryExceeded reports true if current usage has exceeded the limits provided
func (o *observer) memoryExceeded() bool {
	return o.exceeded.Load()
}

func (o *observer) checkMemoryUsage() {
	ms, mismatched := o.stats.Load(o.ballastSize)
	if mismatched && !o.reportedMismatched {
		o.reportedMismatched = true
		o.log.Warn(`"size_mib" in ballast extension is likely incorrectly configured.`)
	}

	o.log.Debug("Current usage memory", memstatToZapField(ms))

	var (
		limitsExceeded = o.exceeded.Load()
		hardExceeded   = o.limits.AboveHardLimit(ms)
		softExceeded   = o.limits.AboveSoftLimit(ms)
	)
	switch {
	case softExceeded, hardExceeded:
		// Since GC is an expensive CPU operation, while we are above the soft limit
		// but the lastGC time is less than the allowed GC interval
		// then it is ignored until the interval has passed or the hard limit is reached.
		if !hardExceeded && o.clock.Since(o.lastGC) < minGCIntervalWhenSoftLimited {
			break
		}
		o.log.Warn("Memory usage has exceeded defined limits, forcing GC event",
			memstatToZapField(ms),
			zap.Bool("soft-breach", softExceeded),
			zap.Bool("hard-breach", hardExceeded),
		)
		o.gc.Do()
		o.lastGC = o.clock.Now()
		o.stats.Update(ms)
	default:
		// In the event that neither limits are being breached currently
		// after previously breaching, the pipeline(s) have returned to safe operation levels.
		if limitsExceeded {
			o.log.Info("Memory usage back within limits. Resuming normal operation.", memstatToZapField(ms))
		}
	}
	// In the event that after a GC operation, the soft limit is still being breached
	// then it should report that data should be dropped until it can recover.
	o.exceeded.Store(o.limits.AboveSoftLimit(ms))
}
