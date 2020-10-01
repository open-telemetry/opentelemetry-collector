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

package groupbytraceprocessor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestEventCallback(t *testing.T) {
	for _, tt := range []struct {
		casename         string
		typ              eventType
		payload          interface{}
		registerCallback func(em *eventMachine, wg *sync.WaitGroup)
	}{
		{
			casename: "onTraceReceived",
			typ:      traceReceived,
			payload:  pdata.NewTraces(),
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onBatchReceived = func(expired pdata.Traces) error {
					wg.Done()
					return nil
				}
			},
		},
		{
			casename: "onTraceExpired",
			typ:      traceExpired,
			payload:  pdata.NewTraceID([]byte{1, 2, 3, 4}),
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceExpired = func(expired pdata.TraceID) error {
					wg.Done()
					assert.Equal(t, pdata.NewTraceID([]byte{1, 2, 3, 4}), expired)
					return nil
				}
			},
		},
		{
			casename: "onTraceReleased",
			typ:      traceReleased,
			payload:  []pdata.ResourceSpans{},
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceReleased = func(expired []pdata.ResourceSpans) error {
					wg.Done()
					return nil
				}
			},
		},
		{
			casename: "onTraceRemoved",
			typ:      traceRemoved,
			payload:  pdata.NewTraceID([]byte{1, 2, 3, 4}),
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceRemoved = func(expired pdata.TraceID) error {
					wg.Done()
					assert.Equal(t, pdata.NewTraceID([]byte{1, 2, 3, 4}), expired)
					return nil
				}
			},
		},
	} {
		t.Run(tt.casename, func(t *testing.T) {
			// prepare
			logger, err := zap.NewDevelopment()
			require.NoError(t, err)

			wg := &sync.WaitGroup{}
			em := newEventMachine(logger, 50)
			tt.registerCallback(em, wg)

			em.startInBackground()
			defer em.shutdown()

			// test
			wg.Add(1)
			em.fire(event{
				typ:     tt.typ,
				payload: tt.payload,
			})

			// verify
			wg.Wait()
		})
	}
}

func TestEventCallbackNotSet(t *testing.T) {
	for _, tt := range []struct {
		casename string
		typ      eventType
	}{
		{
			casename: "onTraceReceived",
			typ:      traceReceived,
		},
		{
			casename: "onTraceExpired",
			typ:      traceExpired,
		},
		{
			casename: "onTraceReleased",
			typ:      traceReleased,
		},
		{
			casename: "onTraceRemoved",
			typ:      traceRemoved,
		},
	} {
		t.Run(tt.casename, func(t *testing.T) {
			// prepare
			logger, err := zap.NewDevelopment()
			require.NoError(t, err)

			wg := &sync.WaitGroup{}
			em := newEventMachine(logger, 50)
			em.onError = func(e event) {
				wg.Done()
			}
			em.startInBackground()
			defer em.shutdown()

			// test
			wg.Add(1)
			em.fire(event{
				typ: tt.typ,
			})

			// verify
			wg.Wait()
		})
	}
}

func TestEventInvalidPayload(t *testing.T) {
	for _, tt := range []struct {
		casename         string
		typ              eventType
		registerCallback func(*eventMachine, *sync.WaitGroup)
	}{
		{
			casename: "onTraceReceived",
			typ:      traceReceived,
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onBatchReceived = func(expired pdata.Traces) error {
					return nil
				}
			},
		},
		{
			casename: "onTraceExpired",
			typ:      traceExpired,
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceExpired = func(expired pdata.TraceID) error {
					return nil
				}
			},
		},
		{
			casename: "onTraceReleased",
			typ:      traceReleased,
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceReleased = func(expired []pdata.ResourceSpans) error {
					return nil
				}
			},
		},
		{
			casename: "onTraceRemoved",
			typ:      traceRemoved,
			registerCallback: func(em *eventMachine, wg *sync.WaitGroup) {
				em.onTraceRemoved = func(expired pdata.TraceID) error {
					return nil
				}
			},
		},
	} {
		t.Run(tt.casename, func(t *testing.T) {
			// prepare
			logger, err := zap.NewDevelopment()
			require.NoError(t, err)

			wg := &sync.WaitGroup{}
			em := newEventMachine(logger, 50)
			em.onError = func(e event) {
				wg.Done()
			}
			tt.registerCallback(em, wg)
			em.startInBackground()
			defer em.shutdown()

			// test
			wg.Add(1)
			em.fire(event{
				typ: tt.typ,
			})

			// verify
			wg.Wait()
		})
	}
}

func TestEventUnknownType(t *testing.T) {
	// prepare
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	em := newEventMachine(logger, 50)
	em.onError = func(e event) {
		wg.Done()
	}
	em.startInBackground()
	defer em.shutdown()

	// test
	wg.Add(1)
	em.fire(event{
		typ: eventType(1234),
	})

	// verify
	wg.Wait()
}

func TestEventShutdown(t *testing.T) {
	// prepare
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	traceReceivedFired, traceExpiredFired := false, false
	em := newEventMachine(logger, 50)
	em.onBatchReceived = func(pdata.Traces) error {
		traceReceivedFired = true
		return nil
	}
	em.onTraceExpired = func(pdata.TraceID) error {
		traceExpiredFired = true
		return nil
	}
	em.onTraceRemoved = func(pdata.TraceID) error {
		wg.Wait()
		return nil
	}
	em.startInBackground()

	// test
	em.fire(event{
		typ:     traceReceived,
		payload: pdata.NewTraces(),
	})
	em.fire(event{
		typ:     traceRemoved,
		payload: pdata.NewTraceID([]byte{1, 2, 3, 4}),
	})
	em.fire(event{
		typ:     traceRemoved,
		payload: pdata.NewTraceID([]byte{1, 2, 3, 4}),
	})

	time.Sleep(10 * time.Millisecond) // give it a bit of time to process the items
	assert.Len(t, em.events, 1)       // we should have one pending event in the queue, the second traceRemoved event

	shutdownWg := sync.WaitGroup{}
	shutdownWg.Add(1)
	go func() {
		em.shutdown()
		shutdownWg.Done()
	}()

	wg.Done()                          // the pending event should be processed
	time.Sleep(100 * time.Millisecond) // give it a bit of time to process the items

	assert.Len(t, em.events, 0)

	// new events should *not* be processed
	em.fire(event{
		typ:     traceExpired,
		payload: pdata.NewTraceID([]byte{1, 2, 3, 4}),
	})

	// verify
	assert.True(t, traceReceivedFired)

	// If the code is wrong, there's a chance that the test will still pass
	// in case the event is processed after the assertion.
	// for this reason, we add a small delay here
	time.Sleep(10 * time.Millisecond)
	assert.False(t, traceExpiredFired)

	// wait until the shutdown has returned
	shutdownWg.Wait()
}

func TestPeriodicMetrics(t *testing.T) {
	// prepare
	views := MetricViews(configtelemetry.LevelDetailed)
	view.Register(views...)
	defer view.Unregister(views...)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	em := newEventMachine(logger, 50)
	em.metricsCollectionInterval = time.Millisecond

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		expected := 2
		calls := 0
		for range em.events {
			// we expect two events, after which we just exit the loop
			// if we return from here, we'd still have one item in the queue that is not going to be consumed
			wg.Wait()
			calls++

			if calls == expected {
				return
			}
		}
	}()

	// sanity check
	assertGaugeNotCreated(t, mNumEventsInQueue)

	// test
	em.fire(event{typ: traceReceived})
	em.fire(event{typ: traceReceived}) // the first is consumed right away, the second is in the queue
	go em.periodicMetrics()

	// ensure our gauge is showing 1 item in the queue
	time.Sleep(10 * time.Millisecond) // wait enough time for the gauge to be updated
	assertGauge(t, 1, mNumEventsInQueue)

	wg.Done() // release all events
	time.Sleep(5 * time.Millisecond)

	// ensure our gauge is now showing no items in the queue
	assertGauge(t, 0, mNumEventsInQueue)

	// signal and wait for the recursive call to finish
	em.shutdownLock.Lock()
	em.closed = true
	em.shutdownLock.Unlock()
	time.Sleep(5 * time.Millisecond)
}

func TestForceShutdown(t *testing.T) {
	// prepare
	em := newEventMachine(logger, 50)
	em.shutdownTimeout = 20 * time.Millisecond

	// test
	em.fire(event{typ: traceExpired})

	start := time.Now()
	em.shutdown() // should take about 20ms to return
	duration := time.Since(start)

	// verify
	assert.True(t, duration > 20*time.Millisecond)
}

func TestDoWithTimeout(t *testing.T) {
	// prepare
	start := time.Now()

	// test
	doWithTimeout(10*time.Millisecond, func() error {
		time.Sleep(20 * time.Second)
		return nil
	})

	// verify
	assert.WithinDuration(t, start, time.Now(), 20*time.Millisecond)
}

func assertGauge(t *testing.T, expected int, gauge *stats.Int64Measure) {
	viewData, err := view.RetrieveData(gauge.Name())
	require.NoError(t, err)
	require.Len(t, viewData, 1) // we expect exactly one data point, the last value

	sum := viewData[0].Data.(*view.LastValueData)
	assert.EqualValues(t, expected, sum.Value)
}

func assertGaugeNotCreated(t *testing.T, gauge *stats.Int64Measure) {
	viewData, err := view.RetrieveData(gauge.Name())
	require.NoError(t, err)
	assert.Len(t, viewData, 0, "gauge exists already but shouldn't")
}
