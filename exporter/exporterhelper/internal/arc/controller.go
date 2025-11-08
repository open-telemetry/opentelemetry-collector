// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/pipeline"
)

type Config struct {
	Enabled        bool    `mapstructure:"enabled"`
	InitialLimit   int     `mapstructure:"initial_limit"`
	MaxConcurrency int     `mapstructure:"max_concurrency"`
	DecreaseRatio  float64 `mapstructure:"decrease_ratio"`
	EwmaAlpha      float64 `mapstructure:"ewma_alpha"`
	DeviationScale float64 `mapstructure:"deviation_scale"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		InitialLimit:   1,
		MaxConcurrency: 200,
		DecreaseRatio:  0.9,
		EwmaAlpha:      0.4,
		DeviationScale: 2.5,
	}
}

type Controller struct {
	cfg Config
	sem *shrinkSem

	mu sync.Mutex
	st state

	// Telemetry
	tel                  *metadata.TelemetryBuilder
	rttInst              metric.Int64Histogram
	failuresInst         metric.Int64Counter
	limitChangesInst     metric.Int64Counter
	limitChangesUpAttr   metric.MeasurementOption
	limitChangesDownAttr metric.MeasurementOption
	backoffInst          metric.Int64Counter
	// common attr set for sync metrics (exporter + data_type)
	commonAttr metric.MeasurementOption
}

type state struct {
	limit       int
	inFlight    int
	past        ewmaVar
	curr        mean
	nextTick    time.Time
	hadPressure bool
	hitCeiling  bool
}

func NewController(cfg Config, tel *metadata.TelemetryBuilder, id component.ID, sig pipeline.Signal) *Controller {
	c := cfg
	if c.InitialLimit <= 0 {
		c.InitialLimit = DefaultConfig().InitialLimit
	}
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = DefaultConfig().MaxConcurrency
	}
	if c.DecreaseRatio <= 0 || c.DecreaseRatio >= 1 {
		c.DecreaseRatio = DefaultConfig().DecreaseRatio
	}
	if c.EwmaAlpha <= 0 || c.EwmaAlpha >= 1 {
		c.EwmaAlpha = DefaultConfig().EwmaAlpha
	}
	if c.DeviationScale < 0 {
		c.DeviationScale = DefaultConfig().DeviationScale
	}

	ctrl := &Controller{
		cfg: c,
		sem: newShrinkSem(c.InitialLimit),
		tel: tel,
	}
	ctrl.st.limit = c.InitialLimit
	ctrl.st.nextTick = time.Now()
	ctrl.st.past = newEwmaVar(c.EwmaAlpha)

	if tel != nil {
		exporterAttr := attribute.String("exporter", id.String())
		dataTypeAttr := attribute.String("data_type", sig.String())

		// Use exporter + data_type on ALL sync datapoints
		common := metric.WithAttributeSet(attribute.NewSet(exporterAttr, dataTypeAttr))
		ctrl.commonAttr = common

		ctrl.rttInst = tel.ExporterArcRttMs
		ctrl.failuresInst = tel.ExporterArcFailures
		ctrl.limitChangesInst = tel.ExporterArcLimitChanges
		ctrl.backoffInst = tel.ExporterArcBackoffEvents

		// Include data_type on the limit change counters too
		ctrl.limitChangesUpAttr = metric.WithAttributeSet(attribute.NewSet(
			exporterAttr, dataTypeAttr, attribute.String("direction", "up"),
		))
		ctrl.limitChangesDownAttr = metric.WithAttributeSet(attribute.NewSet(
			exporterAttr, dataTypeAttr, attribute.String("direction", "down"),
		))

		// Async gauges also include exporter + data_type
		asyncCommon := metric.WithAttributeSet(attribute.NewSet(exporterAttr, dataTypeAttr))
		_ = tel.RegisterExporterArcLimitCallback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(ctrl.CurrentLimit()), asyncCommon)
			return nil
		})
		_ = tel.RegisterExporterArcPermitsInUseCallback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(ctrl.PermitsInUse()), asyncCommon)
			return nil
		})
	}

	return ctrl
}

// Shutdown unregisters any asynchronous callbacks.
func (c *Controller) Shutdown() {
	if c == nil {
		return
	}
	if c.tel != nil {
		c.tel.Shutdown()
	}
	if c.sem != nil {
		c.sem.close()
	}
}

// Acquire blocks until a slot is available or ctx is done.
// Returns true iff the slot was acquired.
func (c *Controller) Acquire(ctx context.Context) bool {
	return c.sem.acquire(ctx) == nil
}

// Release releases one slot back to the semaphore.
func (c *Controller) Release() { c.sem.release() }

// ReleaseWithSample provides feedback to the controller and releases one slot.
func (c *Controller) ReleaseWithSample(ctx context.Context, rtt time.Duration, success, backpressure bool) {
	c.Feedback(ctx, rtt, success, backpressure)
	c.Release()
}

// CurrentLimit returns the current concurrency limit.
func (c *Controller) CurrentLimit() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st.limit
}

func (c *Controller) PermitsInUse() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st.inFlight
}

// StartRequest increments in-flight count and tracks ceiling hit.
func (c *Controller) StartRequest() {
	c.mu.Lock()
	c.st.inFlight++
	if c.st.inFlight >= c.st.limit {
		c.st.hitCeiling = true
	}
	c.mu.Unlock()
}

// Feedback provides the observed RTT and signals if backpressure or success occurred.
func (c *Controller) Feedback(ctx context.Context, rtt time.Duration, success, backpressure bool) {
	now := time.Now()
	c.mu.Lock()

	// Record RTT (guard nil telemetry)
	if c.tel != nil && c.rttInst != nil {
		c.rttInst.Record(ctx, rtt.Milliseconds(), c.commonAttr)
	}

	// Defensively avoid negative inFlight if caller didn't call StartRequest.
	if c.st.inFlight > 0 {
		c.st.inFlight--
	}

	// Track this window's RTTs only for successes (to avoid biasing the mean).
	if success {
		c.st.curr.add(rtt.Seconds())
	}

	// If we got any explicit backpressure signal this window, remember it.
	if backpressure {
		c.st.hadPressure = true
	}

	if !success && c.tel != nil && c.failuresInst != nil {
		// Record failures (non-permanent errors or backpressure).
		c.failuresInst.Add(ctx, 1, c.commonAttr)
	}

	// EARLY BACKOFF ON COLD START:
	// If we've not yet initialized EWMA (no window mean set) but already see pressure,
	// immediately apply multiplicative decrease so we don't stall under load.
	if backpressure && c.st.limit > 1 && !c.st.past.initialized() {
		old := c.st.limit
		n := int(math.Max(1, math.Floor(float64(c.st.limit)*c.cfg.DecreaseRatio)))
		if n < c.st.limit {
			c.sem.forget(c.st.limit - n)
			c.st.limit = n
			if c.tel != nil && c.backoffInst != nil {
				c.backoffInst.Add(ctx, 1, c.commonAttr)
			}
			if c.tel != nil && c.limitChangesInst != nil && c.st.limit != old {
				c.limitChangesInst.Add(ctx, 1, c.limitChangesDownAttr)
			}
		}
	}

	// Initialize the first window based on measured mean.
	if !c.st.past.initialized() {
		if m := c.st.curr.average(); m > 0 {
			c.st.past.update(m)
			// Next tick after 'm' seconds; caller may clamp externally if desired.
			c.st.nextTick = now.Add(time.Duration(m * float64(time.Second)))
		}
		c.mu.Unlock()
		return
	}

	// Window tick: adjust concurrency once per window.
	if !now.Before(c.st.nextTick) {
		m := c.st.curr.average()
		if m > 0 {
			c.st.past.update(m)
		}

		// This will consider hadPressure or an RTT spike vs mean+Kσ.
		c.adjust(ctx, m)

		// Reset window state.
		c.st.curr = mean{}
		c.st.hadPressure = false
		c.st.hitCeiling = false
		c.st.nextTick = now.Add(time.Duration(c.st.past.mean * float64(time.Second)))
	}

	c.mu.Unlock()
}

func (c *Controller) adjust(ctx context.Context, current float64) {
	// NOTE: caller holds c.mu
	dev := math.Sqrt(c.st.past.variance)
	thr := dev * c.cfg.DeviationScale

	// Try additive increase: only when at ceiling, no pressure, and RTT <= mean.
	if c.st.limit < c.cfg.MaxConcurrency &&
		c.st.hitCeiling &&
		!c.st.hadPressure &&
		current > 0 &&
		current <= c.st.past.mean {
		old := c.st.limit
		c.sem.addOne()
		c.st.limit++

		if c.tel != nil && c.limitChangesInst != nil && c.st.limit != old {
			c.limitChangesInst.Add(ctx, 1, c.limitChangesUpAttr)
		}
		return
	}

	// Multiplicative decrease: on explicit pressure or RTT spike.
	if c.st.limit > 1 && (c.st.hadPressure || current >= c.st.past.mean+thr) {
		old := c.st.limit

		n := int(math.Max(1, math.Floor(float64(c.st.limit)*c.cfg.DecreaseRatio)))
		if n < c.st.limit {
			c.sem.forget(c.st.limit - n)
			c.st.limit = n

			// limit decreased => backoff & limit-change callbacks
			if c.tel != nil && c.backoffInst != nil {
				c.backoffInst.Add(ctx, 1, c.commonAttr)
			}
			if c.tel != nil && c.limitChangesInst != nil && c.st.limit != old {
				c.limitChangesInst.Add(ctx, 1, c.limitChangesDownAttr)
			}
		}
	}
}

// ----------------------- shrinkable semaphore -----------------------

type shrinkSem struct {
	mu       sync.Mutex
	avail    int
	waiting  []chan struct{}
	pendingF int
	closed   bool
}

func newShrinkSem(n int) *shrinkSem { return &shrinkSem{avail: n} }

// acquire waits for capacity or returns ctx error.
// NOTE: shrink is paid by future releases (pendingF), not by acquires.
func (s *shrinkSem) acquire(ctx context.Context) error {
	ch := make(chan struct{}, 1)

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return experr.NewShutdownErr(errors.New("arc semaphore closed"))
	}
	if s.avail > 0 {
		s.avail--
		s.mu.Unlock()
		return nil
	}
	s.waiting = append(s.waiting, ch)
	s.mu.Unlock()

	select {
	case <-ch:
		// If we were woken up but are now closed, return error
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			return experr.NewShutdownErr(errors.New("arc semaphore closed"))
		}
		return nil
	case <-ctx.Done():
		// Remove our waiter entry if still queued.
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return experr.NewShutdownErr(errors.New("arc semaphore closed"))
		}
		for i, w := range s.waiting {
			if w == ch {
				s.waiting = append(s.waiting[:i], s.waiting[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		return ctx.Err()
	}
}

// release returns capacity to the semaphore with correct shrink priority:
// 1) pay down pending forgets, 2) wake next waiter, 3) increase avail.
func (s *shrinkSem) release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pendingF > 0 {
		s.pendingF--
		return
	}
	if len(s.waiting) > 0 {
		w := s.waiting[0]
		s.waiting = s.waiting[1:]
		w <- struct{}{}
		return
	}
	s.avail++
}

// forget reduces effective capacity by n.
// It first consumes avail, then accumulates pending forgets to be paid by releases.
func (s *shrinkSem) forget(n int) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < n; i++ {
		if s.avail > 0 {
			s.avail--
		} else {
			s.pendingF++
		}
	}
}

// addOne increases the available permits by exactly 1, paying any pending
// shrink (forget) first, or waking a waiter if present.
func (s *shrinkSem) addOne() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First, pay down any pending shrink.
	if s.pendingF > 0 {
		s.pendingF--
		return
	}

	// If someone is waiting, wake one waiter instead of increasing avail.
	if len(s.waiting) > 0 {
		w := s.waiting[0]
		s.waiting = s.waiting[1:]
		w <- struct{}{}
		return
	}

	// Otherwise just increase available permits.
	s.avail++
}

// close marks the semaphore as closed and wakes up all waiters.
func (s *shrinkSem) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true
	for _, w := range s.waiting {
		close(w) // Close channel to signal
	}
	s.waiting = nil
}
