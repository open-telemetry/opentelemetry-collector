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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	// traces received from the previous processors
	traceReceived eventType = iota

	// traceID to be released
	traceExpired

	// released traces
	traceReleased

	// traceID to be removed
	traceRemoved

	// shutdown
	stop
)

type eventType int
type event struct {
	typ     eventType
	payload interface{}
}

// eventMachine is a simple machine that accepts events in a typically non-blocking manner,
// processing the events serially, to ensure that data at the consumer is consistent.
// Just like the machine itself is non-blocking, consumers are expected to also not block
// on the callbacks, otherwise, events might pile up. When enough events are piled up, firing an
// event will block until enough capacity is available to accept the events.
type eventMachine struct {
	events chan event
	logger *zap.Logger

	onTraceReceived func(pdata.Traces) error
	onTraceExpired  func(pdata.TraceID) error
	onTraceReleased func([]pdata.ResourceSpans) error
	onTraceRemoved  func(pdata.TraceID) error

	onError func(event)

	// shutdown sync
	shutdownLock *sync.RWMutex
	closed       bool
}

func newEventMachine(logger *zap.Logger, bufferSize int) *eventMachine {
	em := &eventMachine{
		logger:       logger,
		events:       make(chan event, bufferSize),
		shutdownLock: &sync.RWMutex{},
	}
	return em
}

func (em *eventMachine) startInBackground() {
	go em.start()
}

func (em *eventMachine) start() {
	for e := range em.events {
		switch e.typ {
		case traceReceived:
			if em.onTraceReceived == nil {
				em.logger.Debug("onTraceReceived not set, skipping event")
				em.callOnError(e)
				continue
			}
			payload, ok := e.payload.(pdata.Traces)
			if !ok {
				// the payload had an unexpected type!
				em.callOnError(e)
				continue
			}
			em.onTraceReceived(payload)
		case traceExpired:
			if em.onTraceExpired == nil {
				em.logger.Debug("onTraceExpired not set, skipping event")
				em.callOnError(e)
				continue
			}
			payload, ok := e.payload.(pdata.TraceID)
			if !ok {
				// the payload had an unexpected type!
				em.callOnError(e)
				continue
			}
			em.onTraceExpired(payload)
		case traceReleased:
			if em.onTraceReleased == nil {
				em.logger.Debug("onTraceReleased not set, skipping event")
				em.callOnError(e)
				continue
			}
			payload, ok := e.payload.([]pdata.ResourceSpans)
			if !ok {
				// the payload had an unexpected type!
				em.callOnError(e)
				continue
			}
			em.onTraceReleased(payload)
		case traceRemoved:
			if em.onTraceRemoved == nil {
				em.logger.Debug("onTraceRemoved not set, skipping event")
				em.callOnError(e)
				continue
			}
			payload, ok := e.payload.(pdata.TraceID)
			if !ok {
				// the payload had an unexpected type!
				em.callOnError(e)
				continue
			}
			em.onTraceRemoved(payload)
		case stop:
			em.logger.Info("shuttting down the event machine")
			em.shutdownLock.Lock()
			em.closed = true
			em.shutdownLock.Unlock()
			e.payload.(*sync.WaitGroup).Done()
			return
		default:
			em.logger.Info("unknown event type", zap.Any("event", e.typ))
			em.callOnError(e)
			continue
		}
	}
}

func (em *eventMachine) fire(events ...event) {
	em.shutdownLock.RLock()
	defer em.shutdownLock.RUnlock()

	// we are not accepting new events
	if em.closed {
		return
	}

	for _, e := range events {
		em.events <- e
	}
}

func (em *eventMachine) shutdown() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	em.events <- event{
		typ:     stop,
		payload: wg,
	}
	wg.Wait()
}

func (em *eventMachine) callOnError(e event) {
	if em.onError != nil {
		em.onError(e)
	}
}
