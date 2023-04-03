// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receivertest // import "go.opentelemetry.io/collector/receiver/receivertest"

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
)

// UniqueIDAttrName is the attribute name that is used in log records/spans/datapoints as the unique identifier.
const UniqueIDAttrName = "test_id"

// UniqueIDAttrDataType is the value type of the UniqueIDAttrName.
type UniqueIDAttrDataType int64

type Generator interface {
	// Generate must generate and send at least one data element (span, log record or metric data point)
	// to the receiver and return a copy of generated element ids.
	// The generated data must contain uniquely identifiable elements, each with a
	// different value of attribute named UniqueIDAttrName.
	// CreateOneLogWithID() can be used a helper to create such logs.
	Generate() IDSet
}

type CheckConsumeContractParams struct {
	T *testing.T
	// Factory that allows to create a receiver.
	Factory receiver.Factory
	// Config of the receiver to use.
	Config component.Config
	// Generator that can send data to the receiver.
	Generator Generator
	// GenerateCount specifies the number of times to call the generator.Generate().
	GenerateCount int
	// ConsumeDecisionFunc defines the decision function to use when the receiver calls Consume() func
	// the next consumer. ConsumeDecisionFunc defines the testing scenario (i.e. to test for
	// success case or for error case or a mix of both). See for example RandomErrorsConsumeDecision.
	ConsumeDecisionFunc consumeDecisionFunc
}

// CheckConsumeContract checks the contract between the receiver and its next consumer. For the contract
// description see ../doc.go. The checker will detect violations of contract on success, on permanent
// and non-permanent errors based on decision-making done by consumeDecisionFunc.
// It is advised to run CheckConsumeContract with a variety of decision-making functions.
func CheckConsumeContract(params CheckConsumeContractParams) {
	consumer := &mockConsumer{t: params.T, consumeDecisionFunc: params.ConsumeDecisionFunc}
	ctx := context.Background()

	// Create and start the receiver.
	receiver, err := params.Factory.CreateLogsReceiver(ctx, NewNopCreateSettings(), params.Config, consumer)
	require.NoError(params.T, err)

	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(params.T, err)

	// Begin generating data to the receiver.

	var generatedIds IDSet
	var generatedIndex int64
	var mux sync.Mutex
	var wg sync.WaitGroup

	const concurrency = 4

	// Create concurrent goroutines that use the generator.
	// The total number of generator calls will be equal to params.GenerateCount.

	for j := 0; j < concurrency; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if atomic.AddInt64(&generatedIndex, 1) >= int64(params.GenerateCount) {
					// Generated as many as was requested. We are done.
					return
				}

				ids := params.Generator.Generate()

				mux.Lock()
				duplicates := generatedIds.Merge(ids)
				mux.Unlock()

				// Check that the generator works correctly. There may not be any duplicates in the
				// generated data set.
				require.Empty(params.T, duplicates)
			}
		}()
	}

	// Wait until all generator goroutines are done.
	wg.Wait()

	// Wait until all data is seen by the consumer.
	assert.Eventually(params.T, func() bool {
		// Calculate the union of accepted and dropped data.
		acceptedAndDropped, duplicates := consumer.acceptedIds.Union(consumer.droppedIds)
		if len(duplicates) != 0 {
			assert.Failf(params.T, "found duplicate elements in received and dropped data", "keys=%v", duplicates)
		}
		// Compare accepted+dropped with generated. Once they are equal it means all data is seen by the consumer.
		missingInOther, onlyInOther := generatedIds.Compare(acceptedAndDropped)
		return len(missingInOther) == 0 && len(onlyInOther) == 0
	}, 5*time.Second, 10*time.Millisecond)

	// Do some final checks. Need the union of accepted and dropped data again.
	acceptedAndDropped, duplicates := consumer.acceptedIds.Union(consumer.droppedIds)
	if len(duplicates) != 0 {
		assert.Failf(params.T, "found duplicate elements in accepted and dropped data", "keys=%v", duplicates)
	}

	// Make sure generated and accepted+dropped are exactly the same.

	missingInOther, onlyInOther := generatedIds.Compare(acceptedAndDropped)
	if len(missingInOther) != 0 {
		assert.Failf(params.T, "found elements sent that were not delivered", "keys=%v", missingInOther)
	}
	if len(onlyInOther) != 0 {
		assert.Failf(params.T, "found elements in accepted and dropped data that was never sent", "keys=%v", onlyInOther)
	}

	err = receiver.Shutdown(ctx)
	assert.NoError(params.T, err)

	// Print some stats to help debug test failures.
	fmt.Printf(
		"Sent %d, accepted=%d, expected dropped=%d, non-permanent errors retried=%d\n",
		len(generatedIds.m),
		len(consumer.acceptedIds.m),
		len(consumer.droppedIds.m),
		consumer.nonPermanentFailures,
	)
}

// IDSet is a set of unique ids of data elements used in the test (logs, spans or metric data points).
type IDSet struct {
	m map[UniqueIDAttrDataType]bool
}

// Compare to another set and calculate the differences from this set.
func (ds *IDSet) Compare(other IDSet) (missingInOther, onlyInOther []UniqueIDAttrDataType) {
	for k := range ds.m {
		if _, ok := other.m[k]; !ok {
			missingInOther = append(missingInOther, k)
		}
	}
	for k := range other.m {
		if _, ok := ds.m[k]; !ok {
			onlyInOther = append(onlyInOther, k)
		}
	}
	return
}

// Merge another set into this one and return a list of duplicate ids.
func (ds *IDSet) Merge(other IDSet) (duplicates []UniqueIDAttrDataType) {
	if ds.m == nil {
		ds.m = map[UniqueIDAttrDataType]bool{}
	}
	for k, v := range other.m {
		if _, ok := ds.m[k]; ok {
			duplicates = append(duplicates, k)
		} else {
			ds.m[k] = v
		}
	}
	return
}

// Union computes the union of this and another sets. A new set if created to return the result.
// Also returns a list of any duplicate ids found.
func (ds *IDSet) Union(other IDSet) (union IDSet, duplicates []UniqueIDAttrDataType) {
	union = IDSet{
		m: map[UniqueIDAttrDataType]bool{},
	}
	for k, v := range ds.m {
		union.m[k] = v
	}
	for k, v := range other.m {
		if _, ok := union.m[k]; ok {
			duplicates = append(duplicates, k)
		} else {
			union.m[k] = v
		}
	}
	return
}

// A function that returns a value indicating what the receiver's next consumer decides
// to do as a result of ConsumeLogs/Trace/Metrics call.
// The result of the decision function becomes the return value of ConsumeLogs/Trace/Metrics.
// Supplying different decision functions allows to test different scenarios of the contract
// between the receiver and it next consumer.
type consumeDecisionFunc func(ids IDSet) error

var errNonPermanent = errors.New("non permanent error")
var errPermanent = errors.New("permanent error")

// RandomNonPermanentErrorConsumeDecision is a decision function that succeeds approximately
// half of the time and fails with a non-permanent error the rest of the time.
func RandomNonPermanentErrorConsumeDecision(_ IDSet) error {
	if rand.Float32() < 0.5 {
		return errNonPermanent
	}
	return nil
}

// RandomPermanentErrorConsumeDecision is a decision function that succeeds approximately
// half of the time and fails with a permanent error the rest of the time.
func RandomPermanentErrorConsumeDecision(_ IDSet) error {
	if rand.Float32() < 0.5 {
		return consumererror.NewPermanent(errPermanent)
	}
	return nil
}

// RandomErrorsConsumeDecision is a decision function that succeeds approximately
// a third of the time, fails with a permanent error the third of the time and fails with
// a non-permanent error the rest of the time.
func RandomErrorsConsumeDecision(_ IDSet) error {
	r := rand.Float64()
	third := 1.0 / 3.0
	if r < third {
		return consumererror.NewPermanent(errPermanent)
	}
	if r < 2*third {
		return errNonPermanent
	}
	return nil
}

// mockConsumer accepts or drops the data from the receiver based on the decision made by
// consumeDecisionFunc and remembers the accepted and dropped data sets for later checks.
// mockConsumer implements all 3 consume functions: ConsumeLogs/ConsumeTraces/ConsumeMetrics
// and can be used for testing any of the 3 signals.
type mockConsumer struct {
	t                    *testing.T
	consumeDecisionFunc  consumeDecisionFunc
	mux                  sync.Mutex
	acceptedIds          IDSet
	droppedIds           IDSet
	nonPermanentFailures int
}

func (m *mockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (m *mockConsumer) ConsumeTraces(_ context.Context, data ptrace.Traces) error {
	ids, err := IDSetFromTraces(data)
	require.NoError(m.t, err)
	return m.consume(ids)
}

// IDSetFromTraces computes an IDSet from given ptrace.Traces. The IDSet will contain ids of all spans.
func IDSetFromTraces(data ptrace.Traces) (IDSet, error) {
	ds := IDSet{
		m: map[UniqueIDAttrDataType]bool{},
	}
	rss := data.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		ils := rss.At(i).ScopeSpans()
		for j := 0; j < ils.Len(); j++ {
			ss := ils.At(j).Spans()
			for k := 0; k < ss.Len(); k++ {
				elem := ss.At(k)
				key, exists := elem.Attributes().Get(UniqueIDAttrName)
				if !exists {
					return ds, fmt.Errorf("invalid data element, attribute %q is missing", UniqueIDAttrName)
				}
				if key.Type() != pcommon.ValueTypeInt {
					return ds, fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
				}
				ds.m[UniqueIDAttrDataType(key.Int())] = true
			}
		}
	}
	return ds, nil
}

func (m *mockConsumer) ConsumeLogs(_ context.Context, data plog.Logs) error {
	ids, err := IDSetFromLogs(data)
	require.NoError(m.t, err)
	return m.consume(ids)
}

// IDSetFromLogs computes an IDSet from given plog.Logs. The IDSet will contain ids of all log records.
func IDSetFromLogs(data plog.Logs) (IDSet, error) {
	ds := IDSet{
		m: map[UniqueIDAttrDataType]bool{},
	}
	rss := data.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		ils := rss.At(i).ScopeLogs()
		for j := 0; j < ils.Len(); j++ {
			ss := ils.At(j).LogRecords()
			for k := 0; k < ss.Len(); k++ {
				elem := ss.At(k)
				key, exists := elem.Attributes().Get(UniqueIDAttrName)
				if !exists {
					return ds, fmt.Errorf("invalid data element, attribute %q is missing", UniqueIDAttrName)
				}
				if key.Type() != pcommon.ValueTypeInt {
					return ds, fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
				}
				ds.m[UniqueIDAttrDataType(key.Int())] = true
			}
		}
	}
	return ds, nil
}

// TODO: Implement mockConsumer.ConsumeMetrics()

// consume the elements with the specified ids, regardless of the element data type.
func (m *mockConsumer) consume(ids IDSet) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Consult with user-defined decision function to decide what to do with the data.
	if err := m.consumeDecisionFunc(ids); err != nil {
		// The decision is to return an error to the receiver.

		if consumererror.IsPermanent(err) {
			// It is a permanent error, which means we need to drop the data.
			// Remember the ids of dropped elements.
			duplicates := m.droppedIds.Merge(ids)
			if len(duplicates) > 0 {
				require.FailNow(m.t, "elements that were dropped previously were sent again: %v", duplicates)
			}
		} else {
			// It is a non-permanent error. Don't add it to the drop list. Remember the number of
			// failures to print at the end of the test.
			m.nonPermanentFailures++
		}
		// Return the error to the receiver.
		return err
	}

	// The decision is a success. Remember the ids of the data in the accepted list.
	duplicates := m.acceptedIds.Merge(ids)
	if len(duplicates) > 0 {
		require.FailNow(m.t, "elements that were accepted previously were sent again: %v", duplicates)
	}
	return nil
}

func CreateOneLogWithID(id UniqueIDAttrDataType) plog.Logs {
	data := plog.NewLogs()
	data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().PutInt(UniqueIDAttrName, int64(id))
	return data
}
