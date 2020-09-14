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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	v1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
)

var (
	logger, _ = zap.NewDevelopment()
)

func TestTraceIsDispatchedAfterDuration(t *testing.T) {
	// prepare
	traces := []*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{
				TraceId: []byte{1, 2, 3, 4}, // no need to be 100% correct here
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
	wg := &sync.WaitGroup{} // we wait for the next (mock) processor to receive the trace

	config := Config{
		// should be long enough for the test to run without traces being finished, but short enough to not
		// badly influence the testing experience
		WaitDuration: 50 * time.Millisecond,

		// we create 6 traces, only 5 should be at the storage in the end
		NumTraces: 5,
	}

	wg.Add(5) // 5 traces are expected to be received

	receivedTraceIDs := []pdata.TraceID{}
	mockProcessor := &mockProcessor{}
	mockProcessor.onTraces = func(ctx context.Context, received pdata.Traces) error {
		traceID := received.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).TraceID()
		receivedTraceIDs = append(receivedTraceIDs, traceID)
		fmt.Println("received trace")
		wg.Done()
		return nil
	}

	st := newMemoryStorage()

	p, err := newGroupByTraceProcessor(logger, st, mockProcessor, config)
	require.NoError(t, err)

	ctx := context.Background()
	p.Start(ctx, nil)
	defer p.Shutdown(ctx)

	// test
	traceIDs := []pdata.TraceID{
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
					TraceId: traceID,
				}},
			}},
		}}

		fmt.Println("sent trace")
		p.ConsumeTraces(ctx, pdata.TracesFromOtlp(batch))
	}

	wg.Wait()

	// verify
	assert.Equal(t, 5, len(receivedTraceIDs))

	for i := 5; i > 0; i-- { // last 5 traces
		traceID := traceIDs[i]
		assert.Contains(t, receivedTraceIDs, traceID)
	}

	// the first trace should have been evicted
	assert.NotContains(t, receivedTraceIDs, traceIDs[0])
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
	emptyRs := pdata.NewResourceSpans()
	batch.ResourceSpans().Append(&emptyRs)

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
		onGet: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, nil
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := []byte{1, 2, 3, 4}
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
	err = p.markAsReleased(traceID)

	// verify
	assert.Error(t, err)
}

func TestTraceErrorFromStorageWhileReleasing(t *testing.T) {
	// prepare
	config := Config{
		WaitDuration: time.Second, // we are not waiting for this whole time
		NumTraces:    5,
	}
	expectedError := errors.New("some unexpected error")
	st := &mockStorage{
		onGet: func(pdata.TraceID) ([]pdata.ResourceSpans, error) {
			return nil, expectedError
		},
	}
	next := &mockProcessor{}

	p, err := newGroupByTraceProcessor(logger, st, next, config)
	require.NoError(t, err)
	require.NotNil(t, p)

	traceID := []byte{1, 2, 3, 4}
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
	err = p.markAsReleased(traceID)

	// verify
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

	traceID := []byte{1, 2, 3, 4}

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

	traceID := []byte{1, 2, 3, 4}

	// test
	err = p.onTraceRemoved(traceID)

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

	traceID := []byte{1, 2, 3, 4}

	// test
	err = p.onTraceRemoved(traceID)

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

	secondTraceID := []byte{2, 3, 4, 5}

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
	firstSpan.SetTraceID([]byte{1, 2, 3, 4})
	secondSpan := firstILS.Spans().At(1)
	secondSpan.SetName("first-batch-second-span")
	secondSpan.SetTraceID([]byte{1, 2, 3, 4})

	// the second ILS has one span
	secondILS := input.InstrumentationLibrarySpans().At(1)
	secondLibrary := secondILS.InstrumentationLibrary()
	secondLibrary.InitEmpty()
	secondLibrary.SetName("second-library")
	secondILS.Spans().Resize(1)
	thirdSpan := secondILS.Spans().At(0)
	thirdSpan.SetName("second-batch-first-span")
	thirdSpan.SetTraceID([]byte{1, 2, 3, 4})

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
	firstSpan.SetTraceID([]byte{1, 2, 3, 4})
	secondSpan := ils.Spans().At(1)
	secondSpan.SetName("first-batch-second-span")
	secondSpan.SetTraceID([]byte{2, 3, 4, 5})

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
	firstSpan.SetTraceID(nil)

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
		traceID := []byte{byte(1 + n), 2, 3, 4}
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
