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
	"math/rand"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/collector/exporter/exportertest"

	"go.opentelemetry.io/collector/internal/data/testdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	v1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"

	_ "net/http/pprof"
)

var (
	logger, _ = zap.NewDevelopment()
)

func TestTraceIsDispatchedAfterDuration(t *testing.T) {
	// prepare
	traces := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				TraceId: otlpcommon.NewTraceID([]byte{1, 2, 3, 4}), // no need to be 100% correct here
			}},
		}},
	}}

	wgReceived := &sync.WaitGroup{} // we wait for the next (mock) processor to receive the trace
	config := Config{
		WaitDuration: time.Nanosecond,
		NumTraces:    10,
	}
	mockProcessor := &mockProcessor{
		onTraces: func(ctx context.Context, received pdata.Traces) error {
			assert.Equal(t, traces, pdata.TracesToOtlp(received))
			wgReceived.Done()
			return nil
		},
	}

	wgDeleted := &sync.WaitGroup{} // we wait for the next (mock) processor to receive the trace
	backing := newMemoryStorage()
	st := &mockStorage{
		onCreateOrAppend: backing.createOrAppend,
		onGet:            backing.get,
		onDelete: func(traceID pdata.TraceID) ([]pdata.ResourceSpans, error) {
			wgDeleted.Done()
			return backing.delete(traceID)
		},
	}

	p, err := newGroupByTraceProcessor(logger, st, mockProcessor, config)
	require.NoError(t, err)

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	// test
	wgReceived.Add(1) // one should be received
	wgDeleted.Add(1)  // one should be deleted
	p.ConsumeTraces(ctx, pdata.TracesFromOtlp(traces))

	// verify
	wgReceived.Wait()
	wgDeleted.Wait()
}

func TestInternalCacheLimit(t *testing.T) {
	// prepare
	config := Config{
		// should be long enough for the test to run without traces being finished, but short enough to not
		// badly influence the testing experience
		WaitDuration: 50 * time.Millisecond,

		// we create 6 traces, only 5 should be at the storage in the end
		NumTraces: 5,
	}

	next := &exportertest.SinkTraceExporter{}

	st := newMemoryStorage()

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)

	timer := manualTimer{funcs: make(chan func(), 100)}
	p.timer = &timer

	ctx := context.Background()
	p.Start(ctx, nil)

	// test
	traceIDs := [][]byte{
		{1, 2, 3, 4},
		{2, 3, 4, 5},
		{3, 4, 5, 6},
		{4, 5, 6, 7},
		{5, 6, 7, 8},
		{6, 7, 8, 9},
	}

	// 6 iterations
	for _, traceID := range traceIDs {
		batch := []*v1.ResourceSpans{{
			InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
				Spans: []*v1.Span{{
					TraceId: otlpcommon.NewTraceID(traceID),
				}},
			}},
		}}

		fmt.Println("sent trace")
		p.ConsumeTraces(ctx, pdata.TracesFromOtlp(batch))
	}

	for i := 0; i < 6; i++ {
		timer.Step()
	}

	p.Shutdown(ctx)

	receivedTraceIDs := []pdata.TraceID{}
	for _, batch := range next.AllTraces() {
		receivedTraceIDs = append(receivedTraceIDs, batch.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).TraceID())
	}

	// verify
	assert.Equal(t, 6, len(receivedTraceIDs))

	for i := 0; i < 6; i++ {
		traceID := pdata.NewTraceID(traceIDs[i])
		assert.Contains(t, receivedTraceIDs, traceID)
	}
}

func TestProcessorCapabilities(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Nanosecond,
		NumTraces:    10,
	}
	st := newMemoryStorage()
	next := &mockProcessor{}

	// test
	p, err := newGroupByTraceProcessor(logger, st, next, config)
	caps := p.GetCapabilities()

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, true, caps.MutatesConsumedData)
}

func TestProcessBatchDoesntFail(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Nanosecond,
		NumTraces:    10,
	}
	st := newMemoryStorage()
	next := &mockProcessor{}

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})

	batch := pdata.NewTraces()
	batch.ResourceSpans().Resize(1)
	trace := batch.ResourceSpans().At(0)
	trace.InstrumentationLibrarySpans().Resize(1)
	ils := trace.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))
	batch.ResourceSpans().Append(pdata.NewResourceSpans())

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)

	// sanity check
	err = p.processResourceSpans(batch.ResourceSpans().At(1))
	require.Error(t, err)

	// test
	err = p.onTraceReceived(batch)

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestTraceDisappearedFromStorageBeforeReleasing(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := &mockStorage{
		onDelete: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, nil
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Create the channel with no buffer so that the test is deterministic
	p.eventMachine.events = make(chan event)

	traceID := otlpcommon.NewTraceID([]byte{1, 2, 3, 4})
	batch := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				TraceId: traceID,
			}},
		}},
	}}

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	err = p.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(batch))
	require.NoError(t, err)

	// test
	// we trigger this manually, instead of waiting the whole duration
	err = p.onTraceExpired(pdata.TraceID(traceID))

	// verify
	assert.Error(t, err)
}

func TestTraceErrorFromStorageWhileReleasing(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: 5 * time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	expectedError := errors.New("some unexpected error")
	st := &mockStorage{
		onDelete: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, expectedError
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Create the channel with no buffer so that the test is deterministic
	p.eventMachine.events = make(chan event)

	traceID := otlpcommon.NewTraceID([]byte{1, 2, 3, 4})
	batch := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				TraceId: traceID,
			}},
		}},
	}}

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	err = p.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(batch))
	require.NoError(t, err)

	// test
	// we trigger this manually, instead of waiting the whole duration
	err = p.onTraceExpired(pdata.TraceID(traceID))

	// verify
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, expectedError))
}

func TestTraceErrorFromStorageWhileProcessingTrace(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	expectedError := errors.New("some unexpected error")
	st := &mockStorage{
		onCreateOrAppend: func(pdata.TraceID, pdata.ResourceSpans) error {
			return expectedError
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})

	rs := pdata.NewResourceSpans()
	rs.InitEmpty()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))

	batch := splitByTrace(rs)

	// test
	err = p.processBatch(batch[0])

	// verify
	assert.True(t, errors.Is(err, expectedError))
}

func TestBadTraceDataWhileProcessingTrace(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := &mockStorage{}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	// test
	err = p.processResourceSpans(pdata.NewResourceSpans())

	// verify
	assert.True(t, errors.Is(err, errNilResourceSpans))
}

func TestAddSpansToExistingTrace(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}
	config := Config{
		WaitDuration: 50 * time.Millisecond,
		NumTraces:    5,
	}
	st := newMemoryStorage()

	receivedTraces := []pdata.ResourceSpans{}
	next := &mockProcessor{
		onTraces: func(ctx context.Context, traces pdata.Traces) error {
			require.Equal(t, 2, traces.ResourceSpans().Len())
			receivedTraces = append(receivedTraces, traces.ResourceSpans().At(0))
			receivedTraces = append(receivedTraces, traces.ResourceSpans().At(1))
			wg.Done()
			return nil
		},
	}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	traceID := otlpcommon.NewTraceID([]byte{1, 2, 3, 4})

	// test
	first := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				Name:    "first-span",
				TraceId: traceID,
			}},
		}},
	}}
	second := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				Name:    "second-span",
				TraceId: traceID,
			}},
		}},
	}}

	wg.Add(1)

	p.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(first))
	p.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(second))

	wg.Wait()

	// verify
	assert.Len(t, receivedTraces, 2)
}

func TestTraceErrorFromStorageWhileProcessingSecondTrace(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := &mockStorage{}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})

	rs := pdata.NewResourceSpans()
	rs.InitEmpty()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))

	batch := splitByTrace(rs)

	// test
	err = p.processBatch(batch[0])
	assert.NoError(t, err)

	expectedError := errors.New("some unexpected error")
	st.onCreateOrAppend = func(pdata.TraceID, pdata.ResourceSpans) error {
		return expectedError
	}

	// processing another batch for the same trace takes a slightly different code path
	err = p.processBatch(batch[0])

	// verify
	assert.True(t, errors.Is(err, expectedError))
}

func TestErrorFromStorageWhileRemovingTrace(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	expectedError := errors.New("some unexpected error")
	st := &mockStorage{
		onDelete: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, expectedError
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})
	p.ringBuffer.put(traceID)

	// test
	err = p.onTraceExpired(traceID)

	// verify
	assert.True(t, errors.Is(err, expectedError))
}

func TestTraceNotFoundWhileRemovingTrace(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := &mockStorage{
		onDelete: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, nil
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})
	p.ringBuffer.put(traceID)

	// test
	err = p.onTraceExpired(traceID)

	// verify
	assert.Error(t, err)
}

func TestTracesAreDispatchedInIndividualBatches(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}

	config := Config{
		WaitDuration: time.Nanosecond, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := newMemoryStorage()
	next := &mockProcessor{
		onTraces: func(_ context.Context, traces pdata.Traces) error {
			// we should receive two batches, each one with one trace
			assert.Equal(t, 1, traces.ResourceSpans().Len())
			wg.Done()
			return nil
		},
	}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	require.NoError(t, err)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})

	firstResourceSpans := pdata.NewResourceSpans()
	firstResourceSpans.InitEmpty()
	firstResourceSpans.InstrumentationLibrarySpans().Resize(1)
	ils := firstResourceSpans.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)

	secondTraceID := pdata.NewTraceID([]byte{2, 3, 4, 5})
	secondResourceSpans := pdata.NewResourceSpans()
	secondResourceSpans.InitEmpty()
	secondResourceSpans.InstrumentationLibrarySpans().Resize(1)
	secondIls := secondResourceSpans.InstrumentationLibrarySpans().At(0)
	secondIls.Spans().Resize(1)
	secondSpan := secondIls.Spans().At(0)
	secondSpan.SetTraceID(secondTraceID)

	// test
	wg.Add(2)

	err = p.processResourceSpans(firstResourceSpans)
	require.NoError(t, err)

	err = p.processResourceSpans(secondResourceSpans)
	require.NoError(t, err)

	wg.Wait()

	// verify
	// verification is done at onTraces from the mockProcessor
}

func TestSplitSameTraceIntoDifferentBatches(t *testing.T) {
	// prepare

	// we have 1 ResourceSpans with 2 ILS, resulting in two batches
	input := pdata.NewResourceSpans()
	input.InitEmpty()
	input.InstrumentationLibrarySpans().Resize(2)

	// the first ILS has two spans
	firstILS := input.InstrumentationLibrarySpans().At(0)
	firstLibrary := firstILS.InstrumentationLibrary()
	firstLibrary.InitEmpty()
	firstLibrary.SetName("first-library")
	firstILS.Spans().Resize(2)
	firstSpan := firstILS.Spans().At(0)
	firstSpan.SetName("first-batch-first-span")
	firstSpan.SetTraceID(pdata.NewTraceID([]byte{1, 2, 3, 4}))
	secondSpan := firstILS.Spans().At(1)
	secondSpan.SetName("first-batch-second-span")
	secondSpan.SetTraceID(pdata.NewTraceID([]byte{1, 2, 3, 4}))

	// the second ILS has one span
	secondILS := input.InstrumentationLibrarySpans().At(1)
	secondLibrary := secondILS.InstrumentationLibrary()
	secondLibrary.InitEmpty()
	secondLibrary.SetName("second-library")
	secondILS.Spans().Resize(1)
	thirdSpan := secondILS.Spans().At(0)
	thirdSpan.SetName("second-batch-first-span")
	thirdSpan.SetTraceID(pdata.NewTraceID([]byte{1, 2, 3, 4}))

	// test
	batches := splitByTrace(input)

	// verify
	assert.Len(t, batches, 2)

	// first batch
	assert.Equal(t, pdata.NewTraceID([]byte{1, 2, 3, 4}), batches[0].traceID)
	assert.Equal(t, firstLibrary.Name(), batches[0].rs.InstrumentationLibrarySpans().At(0).InstrumentationLibrary().Name())
	assert.Equal(t, firstSpan.Name(), batches[0].rs.InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
	assert.Equal(t, secondSpan.Name(), batches[0].rs.InstrumentationLibrarySpans().At(0).Spans().At(1).Name())

	// second batch
	assert.Equal(t, pdata.NewTraceID([]byte{1, 2, 3, 4}), batches[1].traceID)
	assert.Equal(t, secondLibrary.Name(), batches[1].rs.InstrumentationLibrarySpans().At(0).InstrumentationLibrary().Name())
	assert.Equal(t, thirdSpan.Name(), batches[1].rs.InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
}

func TestSplitDifferentTracesIntoDifferentBatches(t *testing.T) {
	// prepare

	// we have 1 ResourceSpans with 1 ILS and two traceIDs, resulting in two batches
	input := pdata.NewResourceSpans()
	input.InitEmpty()
	input.InstrumentationLibrarySpans().Resize(1)

	// the first ILS has two spans
	ils := input.InstrumentationLibrarySpans().At(0)
	library := ils.InstrumentationLibrary()
	library.InitEmpty()
	library.SetName("first-library")
	ils.Spans().Resize(2)
	firstSpan := ils.Spans().At(0)
	firstSpan.SetName("first-batch-first-span")
	firstSpan.SetTraceID(pdata.NewTraceID([]byte{1, 2, 3, 4}))
	secondSpan := ils.Spans().At(1)
	secondSpan.SetName("first-batch-second-span")
	secondSpan.SetTraceID(pdata.NewTraceID([]byte{2, 3, 4, 5}))

	// test
	batches := splitByTrace(input)

	// verify
	assert.Len(t, batches, 2)

	// first batch
	assert.Equal(t, pdata.NewTraceID([]byte{1, 2, 3, 4}), batches[0].traceID)
	assert.Equal(t, library.Name(), batches[0].rs.InstrumentationLibrarySpans().At(0).InstrumentationLibrary().Name())
	assert.Equal(t, firstSpan.Name(), batches[0].rs.InstrumentationLibrarySpans().At(0).Spans().At(0).Name())

	// second batch
	assert.Equal(t, pdata.NewTraceID([]byte{2, 3, 4, 5}), batches[1].traceID)
	assert.Equal(t, library.Name(), batches[1].rs.InstrumentationLibrarySpans().At(0).InstrumentationLibrary().Name())
	assert.Equal(t, secondSpan.Name(), batches[1].rs.InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
}

func TestSplitByTraceWithNilTraceID(t *testing.T) {
	// prepare
	input := pdata.NewResourceSpans()
	input.InitEmpty()
	input.InstrumentationLibrarySpans().Resize(1)
	ils := input.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	firstSpan := ils.Spans().At(0)
	firstSpan.SetTraceID(pdata.NewTraceID(nil))

	// test
	batches := splitByTrace(input)

	// verify
	assert.Len(t, batches, 0)
}

func TestErrorOnProcessResourceSpansContinuesProcessing(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	st := &mockStorage{}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})

	rs := pdata.NewResourceSpans()
	rs.InitEmpty()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))

	expectedError := errors.New("some unexpected error")
	returnedError := false
	st.onCreateOrAppend = func(pdata.TraceID, pdata.ResourceSpans) error {
		returnedError = true
		return expectedError
	}

	// test
	err = p.processResourceSpans(rs)

	// verify
	assert.NoError(t, err)
	assert.True(t, returnedError)
}

func BenchmarkConsumeTracesCompleteOnFirstBatch(b *testing.B) {
	// prepare
	config := Config{
		WaitDuration: 50 * time.Millisecond,
		NumTraces:    defaultNumTraces,
	}
	st := newMemoryStorage()
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(zap.NewNop(), st, next, config)
	require.NoError(b, err)
	require.NotNil(b, p)

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	for n := 0; n < b.N; n++ {
		traceID := otlpcommon.NewTraceID([]byte{byte(1 + n), 2, 3, 4})
		trace := []*v1.ResourceSpans{{
			InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
				Spans: []*v1.Span{{
					TraceId: traceID,
				}},
			}},
		}}
		p.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(trace))
	}
}

type concurrencyTest struct {
	name           string
	numTraces      int
	spansPerTrace  int
	ringBufferSize int
	waitDuration   time.Duration
}

func TestHighConcurrency(t *testing.T) {

	tests := []concurrencyTest{
		{
			name:           "exceed_ring_buffer_size",
			numTraces:      400,
			spansPerTrace:  10,
			ringBufferSize: 100,
			waitDuration:   100 * time.Millisecond,
		},
		{
			name:           "fast_wait_duration",
			numTraces:      400,
			spansPerTrace:  10,
			ringBufferSize: 10000,
			waitDuration:   1 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			traceIds, batches := generateTraces(test.numTraces, test.spansPerTrace)
			// Shuffle the batches so that the spans for each trace will arrive in a random order
			rand.Shuffle(len(batches), func(i, j int) { batches[i], batches[j] = batches[j], batches[i] })

			expectedSpanCount := 0
			for _, b := range batches {
				expectedSpanCount += b.SpanCount()
			}
			require.EqualValues(t, test.numTraces*test.spansPerTrace, expectedSpanCount)

			st := newMemoryStorage()
			next := &exportertest.SinkTraceExporter{}

			config := Config{
				WaitDuration: test.waitDuration,
				NumTraces:    test.ringBufferSize,
			}

			logger := zap.NewNop()
			// For local debugging
			//conf := zap.NewDevelopmentConfig()
			//// Debug to help follow the operations on the trace
			//conf.Level.SetLevel(zapcore.DebugLevel)
			//logger, err := conf.Build()
			//require.NoError(t, err)

			p, err := newGroupByTraceProcessor(logger, st, next, config)
			require.NoError(t, err)
			// swap the timer for an implementation we can wait on
			timer := waitableTimer{}
			p.timer = &timer

			ctx := context.Background()
			p.Start(ctx, nil)

			wg := sync.WaitGroup{}
			wg.Add(len(batches))

			for _, b := range batches {
				go func(batch pdata.Traces) {
					_ = p.ConsumeTraces(context.Background(), batch)
					wg.Done()
				}(b)
			}

			// Wait until all calls to ConsumeTraces have completed
			wg.Wait()

			// Wait for all events to be emitted by the timer
			timer.wg.Wait()

			// All events have been emitted, this will wait until they are all consumed
			p.Shutdown(ctx)

			receivedTraceBatches := next.AllTraces()
			// ideally this would equal len(traceIds) but its not always the case b/c of timing races of the enqueue/dequeue
			uniqTraceIdsToSpanCount := map[string]int{}
			for _, batch := range receivedTraceBatches {
				rs := batch.ResourceSpans()
				for i := 0; i < rs.Len(); i++ {
					ils := rs.At(i).InstrumentationLibrarySpans()
					for k := 0; k < ils.Len(); k++ {
						spans := ils.At(k).Spans()
						for s := 0; s < ils.Len(); s++ {
							traceId := spans.At(s).TraceID().HexString()
							if _, ok := uniqTraceIdsToSpanCount[traceId]; !ok {
								uniqTraceIdsToSpanCount[traceId] = 0
							}
							uniqTraceIdsToSpanCount[traceId] = uniqTraceIdsToSpanCount[traceId] + 1
						}
					}
				}
			}

			assert.EqualValues(t, len(traceIds), len(uniqTraceIdsToSpanCount))

			// Under the current buffer eviction code path it intentionally discards spans, but that seems wrong
			// There is a separate bug in the release phase where the operation isn't atomic and spans can be release multiple times
			for k, v := range uniqTraceIdsToSpanCount {
				assert.EqualValues(t, test.spansPerTrace, v, "Trace %s should have %d spans", k, test.spansPerTrace)
			}

			assert.EqualValues(t, expectedSpanCount, next.SpansCount())
		})
	}
}

func generateTraces(numTraces int, numSpans int) ([][]byte, []pdata.Traces) {
	traceIds := make([][]byte, numTraces)
	var tds []pdata.Traces
	for i := 0; i < numTraces; i++ {
		traceIds[i] = tracetranslator.UInt64ToByteTraceID(1, uint64(i+1))
		// Send each span in a separate batch
		for j := 0; j < numSpans; j++ {
			td := testdata.GenerateTraceDataOneSpan()
			span := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
			span.SetTraceID(pdata.NewTraceID(traceIds[i]))
			span.SetSpanID(tracetranslator.UInt64ToByteSpanID(uint64(i<<5) + uint64(j+1)))
			tds = append(tds, td)
		}
	}

	return traceIds, tds
}

type mockProcessor struct {
	onTraces func(context.Context, pdata.Traces) error
}

var _ component.TraceProcessor = (*mockProcessor)(nil)

func (m *mockProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if m.onTraces != nil {
		return m.onTraces(ctx, td)
	}
	return nil
}
func (m *mockProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}
func (m *mockProcessor) Shutdown(context.Context) error {
	return nil
}
func (m *mockProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

type mockStorage struct {
	onCreateOrAppend func(pdata.TraceID, pdata.ResourceSpans) error
	onGet            func(pdata.TraceID) ([]pdata.ResourceSpans, error)
	onDelete         func(pdata.TraceID) ([]pdata.ResourceSpans, error)
}

var _ storage = (*mockStorage)(nil)

func (st *mockStorage) createOrAppend(traceID pdata.TraceID, trace pdata.ResourceSpans) error {
	if st.onCreateOrAppend != nil {
		return st.onCreateOrAppend(traceID, trace)
	}
	return nil
}
func (st *mockStorage) get(traceID pdata.TraceID) ([]pdata.ResourceSpans, error) {
	if st.onGet != nil {
		return st.onGet(traceID)
	}
	return nil, nil
}
func (st *mockStorage) delete(traceID pdata.TraceID) ([]pdata.ResourceSpans, error) {
	if st.onDelete != nil {
		return st.onDelete(traceID)
	}
	return nil, nil
}

type waitableTimer struct {
	wg sync.WaitGroup
}

func (t *waitableTimer) AfterFunc(d time.Duration, f func()) {
	t.wg.Add(1)
	time.AfterFunc(d, func() {
		defer t.wg.Done()
		f()
	})
}

type manualTimer struct {
	funcs chan func()
}

func (t *manualTimer) AfterFunc(d time.Duration, f func()) {
	t.funcs <- f
}

func (t *manualTimer) Step() {
	f := <-t.funcs
	f()
}
