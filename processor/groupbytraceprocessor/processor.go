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
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var (
	errNilResourceSpans = errors.New("invalid resource spans (nil)")
)

// groupByTraceProcessor is a processor that keeps traces in memory for a given duration, with the expectation
// that the trace will be complete once this duration expires. After the duration, the trace is sent to the next consumer.
// This processor uses a buffered event machine, which converts operations into events for non-blocking processing, but
// keeping all operations serialized. This ensures that we don't need locks but that the state is consistent across go routines.
// Each in-flight trace is registered with a go routine, which will be called after the given duration and dispatched to the event
// machine for further processing.
// The typical data flow looks like this:
// ConsumeTraces -> event(traceReceived) -> onTraceReceived -> AfterFunc(duration, event(traceExpired)) -> onTraceExpired
// async markAsReleased -> event(traceReleased) -> onTraceReleased -> nextConsumer
// This processor uses also a ring buffer to hold the in-flight trace IDs, so that we don't hold more than the given maximum number
// of traces in memory/storage. Items that are evicted from the buffer are discarded without warning.
type groupByTraceProcessor struct {
	nextConsumer consumer.TraceConsumer
	config       Config
	logger       *zap.Logger

	// the event machine handling all operations for this processor
	eventMachine *eventMachine

	// the ring buffer holding the IDs for all the in-flight traces
	ringBuffer *ringBuffer

	// the trace storage
	st storage
}

var _ component.TraceProcessor = (*groupByTraceProcessor)(nil)

// newGroupByTraceProcessor returns a new processor.
func newGroupByTraceProcessor(logger *zap.Logger, st storage, nextConsumer consumer.TraceConsumer, config Config) (*groupByTraceProcessor, error) {
	// the event machine will buffer up to 200 concurrent events before blocking
	eventMachine := newEventMachine(logger, 200)

	sp := &groupByTraceProcessor{
		logger:       logger,
		nextConsumer: nextConsumer,
		config:       config,
		eventMachine: eventMachine,
		ringBuffer:   newRingBuffer(config.NumTraces),
		st:           st,
	}

	// register the callbacks
	eventMachine.onTraceReceived = sp.onTraceReceived
	eventMachine.onTraceExpired = sp.onTraceExpired

	return sp, nil
}

func (sp *groupByTraceProcessor) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	sp.eventMachine.fire(event{
		typ:     traceReceived,
		payload: td,
	})
	return nil
}

func (sp *groupByTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (sp *groupByTraceProcessor) Start(context.Context, component.Host) error {
	sp.eventMachine.startInBackground()
	return nil
}

// Shutdown is invoked during service shutdown.
func (sp *groupByTraceProcessor) Shutdown(_ context.Context) error {
	sp.eventMachine.shutdown()
	return nil
}

func (sp *groupByTraceProcessor) onTraceReceived(batch pdata.Traces) error {
	for i := 0; i < batch.ResourceSpans().Len(); i++ {
		if err := sp.processResourceSpans(batch.ResourceSpans().At(i)); err != nil {
			sp.logger.Info("failed to process trace", zap.Error(err))
		}
	}

	return nil
}

func (sp *groupByTraceProcessor) processResourceSpans(rs pdata.ResourceSpans) error {
	if rs.IsNil() {
		// should not happen with the current code
		return errNilResourceSpans
	}

	for _, batch := range splitByTrace(rs) {
		if err := sp.processBatch(batch); err != nil {
			sp.logger.Info("failed to process batch", zap.Error(err),
				zap.String("traceID", batch.traceID.HexString()))
		}
	}

	return nil
}

func (sp *groupByTraceProcessor) processBatch(batch *singleTraceBatch) error {
	traceID := batch.traceID
	if sp.ringBuffer.contains(traceID) {
		// it exists in memory already, just append the spans to the trace in the storage
		if err := sp.addSpans(traceID, batch.rs); err != nil {
			return fmt.Errorf("couldn't add spans to existing trace: %w", err)
		}

		// we are done with this trace, move on
		return nil
	}

	// at this point, we determined that we haven't seen the trace yet, so, record the
	// traceID in the map and the spans to the storage

	// place the trace ID in the buffer, and check if an item had to be evicted
	evicted := sp.ringBuffer.put(traceID)
	if evicted.Bytes() != nil {
		// TODO: do we want another channel that receives evicted items? record a metric perhaps?
		sp.logger.Info("trace evicted: in order to avoid this in the future, adjust the wait duration and/or number of traces to keep in memory",
			zap.String("traceID", evicted.HexString()))
		// delete from the storage
		trace, err := sp.st.delete(evicted)
		if err != nil {
			sp.logger.Info(fmt.Sprintf("couldn't delete trace %q from the storage: %s", traceID.HexString(), err.Error()))
		} else if trace == nil {
			sp.logger.Info(fmt.Sprintf("trace %q not found at the storage", traceID.HexString()))
		} else {
			// no need to wait on the nextConsumer
			go sp.releaseTrace(evicted, trace)
		}
	}

	// we have the traceID in the memory, place the spans in the storage too
	if err := sp.addSpans(traceID, batch.rs); err != nil {
		return fmt.Errorf("couldn't add spans to new trace: %w", err)
	}

	sp.logger.Debug("scheduled to release trace", zap.Duration("duration", sp.config.WaitDuration))

	time.AfterFunc(sp.config.WaitDuration, func() {
		// if the event machine has stopped, it will just discard the event
		sp.eventMachine.fire(event{
			typ:     traceExpired,
			payload: traceID,
		})
	})

	return nil
}

func (sp *groupByTraceProcessor) onTraceExpired(traceID pdata.TraceID) error {
	sp.logger.Debug("processing expired", zap.String("traceID",
		traceID.HexString()))

	if !sp.ringBuffer.contains(traceID) {
		// we likely received multiple batches with spans for the same trace
		// and released this trace already
		sp.logger.Debug("skipping the processing of expired trace",
			zap.String("traceID", traceID.HexString()))
		return nil
	}

	// delete from the map and erase its memory entry
	sp.ringBuffer.delete(traceID)

	sp.logger.Debug("marking the trace as released",
		zap.String("traceID", traceID.HexString()))

	trace, err := sp.st.delete(traceID)
	if err != nil {
		return fmt.Errorf("couldn't delete trace %q from the storage: %w", traceID.HexString(), err)
	}

	if trace == nil {
		return fmt.Errorf("trace %q not found at the storage", traceID.HexString())
	}
	// no need to wait on the nextConsumer
	go sp.releaseTrace(traceID, trace)

	return nil
}

func (sp *groupByTraceProcessor) releaseTrace(traceId pdata.TraceID, rss []pdata.ResourceSpans) {
	trace := pdata.NewTraces()
	for _, rs := range rss {
		trace.ResourceSpans().Append(rs)
	}

	if err := sp.nextConsumer.ConsumeTraces(context.Background(), trace); err != nil {
		sp.logger.Debug("next processor failed to process trace", zap.Error(err))
	}
}

func (sp *groupByTraceProcessor) addSpans(traceID pdata.TraceID, trace pdata.ResourceSpans) error {
	sp.logger.Debug("creating trace at the storage", zap.String("traceID", traceID.HexString()))
	return sp.st.createOrAppend(traceID, trace)
}

type singleTraceBatch struct {
	traceID pdata.TraceID
	rs      pdata.ResourceSpans
}

func splitByTrace(rs pdata.ResourceSpans) []*singleTraceBatch {
	// for each span in the resource spans, we group them into batches of rs/ils/traceID.
	// if the same traceID exists in different ils, they land in different batches.
	var result []*singleTraceBatch

	for i := 0; i < rs.InstrumentationLibrarySpans().Len(); i++ {
		// the batches for this ILS
		batches := map[string]*singleTraceBatch{}

		ils := rs.InstrumentationLibrarySpans().At(i)
		for j := 0; j < ils.Spans().Len(); j++ {
			span := ils.Spans().At(j)
			if span.TraceID().Bytes() == nil {
				// this should have already been caught before our processor, but let's
				// protect ourselves against bad clients
				continue
			}

			sTraceID := span.TraceID().HexString()

			// for the first traceID in the ILS, initialize the map entry
			// and add the singleTraceBatch to the result list
			if _, ok := batches[sTraceID]; !ok {
				newRS := pdata.NewResourceSpans()
				newRS.InitEmpty()
				// currently, the ResourceSpans implementation has only a Resource and an ILS. We'll copy the Resource
				// and set our own ILS
				rs.Resource().CopyTo(newRS.Resource())

				newILS := pdata.NewInstrumentationLibrarySpans()
				newILS.InitEmpty()
				// currently, the ILS implementation has only an InstrumentationLibrary and spans. We'll copy the library
				// and set our own spans
				ils.InstrumentationLibrary().CopyTo(newILS.InstrumentationLibrary())
				newRS.InstrumentationLibrarySpans().Append(newILS)

				batch := &singleTraceBatch{
					traceID: span.TraceID(),
					rs:      newRS,
				}
				batches[sTraceID] = batch
				result = append(result, batch)
			}

			// there is only one instrumentation library per batch
			batches[sTraceID].rs.InstrumentationLibrarySpans().At(0).Spans().Append(span)
		}
	}

	return result
}
