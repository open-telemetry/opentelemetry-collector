// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/pipeline"
)

// newTestController creates a controller with NOP telemetry.
func newTestController(t *testing.T, cfg Config) *Controller {
	set := componenttest.NewNopTelemetrySettings()
	tel, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)
	return NewController(cfg, tel, component.MustNewID("test"), pipeline.SignalTraces)
}

// forceControlStep triggers a feedback loop that is guaranteed to be after the period duration.
func forceControlStep(c *Controller) {
	// Wait past the period and poke again to trigger controlStep
	time.Sleep(c.st.periodDur + 1*time.Millisecond)
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), time.Duration(c.st.lastRTTMean*float64(time.Millisecond)), true, false)
}

func TestController_NewShutdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	c := newTestController(t, cfg)
	require.NotNil(t, c)
	assert.Equal(t, cfg.InitialLimit, c.CurrentLimit())
	c.Shutdown()
}

func TestController_ConfigClamping(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	// Intentionally bad values that should be clamped to defaults
	cfg.InitialLimit = -5
	cfg.MaxConcurrency = 0
	cfg.DecreaseRatio = 1.5
	cfg.EwmaAlpha = -1.0
	cfg.DeviationScale = -2.0

	c := newTestController(t, cfg)
	require.NotNil(t, c)

	// NewController should have clamped these per DefaultConfig()+rules
	def := DefaultConfig()
	assert.GreaterOrEqual(t, c.cfg.InitialLimit, 1)
	assert.Equal(t, def.MaxConcurrency, c.cfg.MaxConcurrency)
	assert.InDelta(t, def.DecreaseRatio, c.cfg.DecreaseRatio, 1e-9)
	assert.InDelta(t, def.EwmaAlpha, c.cfg.EwmaAlpha, 1e-9)
	assert.InDelta(t, def.DeviationScale, c.cfg.DeviationScale, 1e-9)

	// Also ensure InitialLimit <= MaxConcurrency
	assert.LessOrEqual(t, c.cfg.InitialLimit, c.cfg.MaxConcurrency)
	c.Shutdown()
}

func TestController_AcquireRelease(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 2
	c := newTestController(t, cfg)
	require.NotNil(t, c)

	// Acquire two permits
	assert.True(t, c.Acquire(context.Background()))
	assert.Equal(t, 1, c.PermitsInUse())
	assert.True(t, c.Acquire(context.Background()))
	assert.Equal(t, 2, c.PermitsInUse())

	// Third acquire should block and fail with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assert.False(t, c.Acquire(ctx))
	assert.Equal(t, 2, c.PermitsInUse())

	// Release one and acquire again
	c.Release()
	assert.Equal(t, 1, c.PermitsInUse())
	assert.True(t, c.Acquire(context.Background()))
	assert.Equal(t, 2, c.PermitsInUse())

	c.Shutdown()
}

func TestController_AcquireRespectsContextCancel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 1

	c := newTestController(t, cfg)
	require.NotNil(t, c)

	// Take the only permit
	assert.True(t, c.Acquire(context.Background()))

	// Second acquire should obey context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	ok := c.Acquire(ctx)
	assert.False(t, ok)

	// Cleanup
	c.Release()
	c.Shutdown()
}

func TestController_StartRequestCreditsOnlyOnSaturation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 2

	c := newTestController(t, cfg)

	// Force a short period so control steps run frequently in other tests,
	// but here we're just testing credit accrual behavior.
	c.mu.Lock()
	c.st.periodDur = 10 * time.Millisecond
	c.mu.Unlock()

	// At start, no in-flight and no credits.
	assert.Equal(t, 0, c.st.inFlight)
	assert.Equal(t, 0, c.st.credits)

	// First request (inFlight=1 < limit=2) -> no credit.
	c.StartRequest()
	assert.Equal(t, 1, c.st.inFlight)
	assert.Equal(t, 0, c.st.credits)

	// Second request (inFlight=2 == limit=2) -> credit++.
	c.StartRequest()
	assert.Equal(t, 2, c.st.inFlight)
	assert.Equal(t, 1, c.st.credits)

	// Third request (inFlight=3 > limit=2) -> credit++.
	c.StartRequest()
	assert.Equal(t, 3, c.st.inFlight)
	assert.Equal(t, 2, c.st.credits)

	// Release all three via ReleaseWithSample which also calls Release().
	c.ReleaseWithSample(context.Background(), 10*time.Millisecond, true, false)
	c.ReleaseWithSample(context.Background(), 10*time.Millisecond, true, false)
	c.ReleaseWithSample(context.Background(), 10*time.Millisecond, true, false)
	assert.Equal(t, 0, c.st.inFlight)

	c.Shutdown()
}

func TestController_Feedback_UpdatesEWMAOnlyOnSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 1
	cfg.EwmaAlpha = 0.5

	c := newTestController(t, cfg)

	// First successful sample initializes the robust EWMA
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 100*time.Millisecond, true, false)
	require.True(t, c.st.reMean.initialized())
	mean1 := c.st.lastRTTMean
	dev1 := c.st.lastRTTDev

	// Failed/backpressure=false sample should NOT update EWMA
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 400*time.Millisecond, false, false)
	assert.InDelta(t, mean1, c.st.lastRTTMean, 1e-9)
	assert.InDelta(t, dev1, c.st.lastRTTDev, 1e-9)
	// Successful sample should update EWMA
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 200*time.Millisecond, true, false)
	assert.Greater(t, math.Abs(mean1-c.st.lastRTTMean), 1e-9)

	c.Shutdown()
}

func TestController_AdditiveIncreaseAndCreditReset(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 1
	cfg.MaxConcurrency = 5
	cfg.EwmaAlpha = 0.5
	cfg.DeviationScale = 2.0

	c := newTestController(t, cfg)
	// Short test period
	c.mu.Lock()
	c.st.periodDur = 10 * time.Millisecond
	c.mu.Unlock()

	// Prime EWMA with a stable RTT (so no spike)
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 100*time.Millisecond, true, false)

	// Force a control step to set prevRTTMean
	forceControlStep(c)
	assert.Equal(t, 1, c.CurrentLimit()) // Should not have increased
	assert.Positive(t, c.st.prevRTTMean)

	// Build up enough credits to trigger an increase.
	reqCredits := requiredCredits(c.CurrentLimit())
	for i := 0; i < reqCredits; i++ {
		c.StartRequest()
	}
	// Release these; we want the control step to see saturation and no pressure.
	for i := 0; i < reqCredits; i++ {
		c.ReleaseWithSample(context.Background(), 100*time.Millisecond, true, false)
	}

	// Wait past the period and poke again to trigger controlStep
	forceControlStep(c)

	assert.Equal(t, 2, c.CurrentLimit(), "limit should increase additively by 1")
	assert.Equal(t, 0, c.st.credits, "credits reset after increase")

	c.Shutdown()
}

func TestController_MultiplicativeDecreaseOnBackpressure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 4
	cfg.MaxConcurrency = 10
	cfg.DecreaseRatio = 0.5 // make the effect obvious

	c := newTestController(t, cfg)
	require.Equal(t, 4, c.CurrentLimit())

	// Short control period
	c.mu.Lock()
	c.st.periodDur = 10 * time.Millisecond
	c.mu.Unlock()

	// Cause an explicit backpressure signal
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 50*time.Millisecond, true, true)

	// Wait and trigger the control step
	forceControlStep(c)

	// newLimit = floor(4 * 0.5) = 2
	assert.Equal(t, 2, c.CurrentLimit())
	assert.Equal(t, 0, c.st.credits)

	c.Shutdown()
}

func TestController_DecreaseOnRTTSpike(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 4
	cfg.DeviationScale = 1.0
	cfg.EwmaAlpha = 0.5

	c := newTestController(t, cfg)

	// Short control period
	c.mu.Lock()
	c.st.periodDur = 10 * time.Millisecond
	c.mu.Unlock()

	// Establish baseline RTT ~100ms
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 100*time.Millisecond, true, false)

	// Force a control step to set prevRTTMean
	forceControlStep(c)
	assert.Equal(t, 4, c.CurrentLimit())
	assert.Positive(t, c.st.prevRTTMean)

	// Trigger a spike sample
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 600*time.Millisecond, true, false)

	// After control step, limit should drop by DecreaseRatio (default 0.9)
	forceControlStep(c)

	// Expect a decrease by floor(4*0.9) = 3 (one step down)
	assert.Equal(t, 3, c.CurrentLimit())
	c.Shutdown()
}

func TestController_MinFloorAndMaxCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 1
	cfg.MaxConcurrency = 2
	cfg.DecreaseRatio = 0.1

	c := newTestController(t, cfg)

	// Try to decrease at the floor; should stay at 1
	c.mu.Lock()
	c.st.periodDur = 10 * time.Millisecond
	c.mu.Unlock()

	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 50*time.Millisecond, true, true) // pressure
	forceControlStep(c)
	assert.Equal(t, 1, c.CurrentLimit(), "should not go below 1")

	// Build credits to increase to the cap = 2
	reqCredits := requiredCredits(c.CurrentLimit())
	for i := 0; i < reqCredits; i++ {
		c.StartRequest()
	}
	for i := 0; i < reqCredits; i++ {
		c.ReleaseWithSample(context.Background(), 50*time.Millisecond, true, false)
	}
	forceControlStep(c)
	assert.Equal(t, 2, c.CurrentLimit(), "should increase to cap")

	// Further credit should not exceed MaxConcurrency
	reqCredits = requiredCredits(c.CurrentLimit())
	for i := 0; i < reqCredits; i++ {
		c.StartRequest()
	}
	for i := 0; i < reqCredits; i++ {
		c.ReleaseWithSample(context.Background(), 50*time.Millisecond, true, false)
	}
	forceControlStep(c)
	assert.Equal(t, 2, c.CurrentLimit(), "should not exceed cap")

	c.Shutdown()
}

func TestController_ReleaseWithSampleReleases(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.InitialLimit = 1

	c := newTestController(t, cfg)

	// Acquire via the pool to bump inUse
	assert.True(t, c.Acquire(context.Background()))
	assert.Equal(t, 1, c.PermitsInUse())

	// StartRequest increments inFlight; ReleaseWithSample must also release the pool permit
	c.StartRequest()
	c.ReleaseWithSample(context.Background(), 10*time.Millisecond, true, false)

	assert.Equal(t, 0, c.PermitsInUse())
	c.Shutdown()
}

func Test_requiredCredits(t *testing.T) {
	assert.Equal(t, 1, requiredCredits(0))
	assert.Equal(t, 1, requiredCredits(1))
	assert.Equal(t, 2, requiredCredits(2))
	assert.Equal(t, 2, requiredCredits(7))
	assert.Equal(t, 3, requiredCredits(8))
	assert.Equal(t, 3, requiredCredits(31))
	assert.Equal(t, 4, requiredCredits(32))
	assert.Equal(t, 4, requiredCredits(100))
}

func Test_contextOrBG(t *testing.T) {
	var nilCtx context.Context // avoid SA1012 by not passing a literal nil
	bg := contextOrBG(nilCtx)
	require.NotNil(t, bg)

	ctx := context.Background()
	require.Equal(t, ctx, contextOrBG(ctx))
}
