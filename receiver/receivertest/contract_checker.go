// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

// UniqueIDAttrVal is the value type of the UniqueIDAttrName.
type UniqueIDAttrVal string

type Generator interface {
	// Start the generator and prepare to generate. Will be followed by calls to Generate().
	// Start() may be called again after Stop() is called to begin a new test scenario.
	Start()

	// Stop generating. There will be no more calls to Generate() until Start() is called again.
	Stop()

	// Generate must generate and send at least one data element (span, log record or metric data point)
	// to the receiver and return a copy of generated element ids.
	// The generated data must contain uniquely identifiable elements, each with a
	// different value of attribute named UniqueIDAttrName.
	// CreateOneLogWithID() can be used a helper to create such logs.
	// May be called concurrently from multiple goroutines.
	Generate() []UniqueIDAttrVal
}

type CheckConsumeContractParams struct {
	T *testing.T
	// Factory that allows to create a receiver.
	Factory receiver.Factory
	// DataType to test for.
	DataType component.DataType
	// Config of the receiver to use.
	Config component.Config
	// Generator that can send data to the receiver.
	Generator Generator
	// GenerateCount specifies the number of times to call the generator.Generate()
	// for each test scenario.
	GenerateCount int
}

// CheckConsumeContract checks the contract between the receiver and its next consumer. For the contract
// description see ../doc.go. The checker will detect violations of contract on different scenarios: on success,
// on permanent and non-permanent errors and mix of error types.
func CheckConsumeContract(params CheckConsumeContractParams) {
	// Different scenarios to test for.
	// The decision function defines the testing scenario (i.e. to test for
	// success case or for error case or a mix of both). See for example randomErrorsConsumeDecision.
	scenarios := []struct {
		name         string
		decisionFunc func(ids idSet) error
	}{
		{
			name: "always_succeed",
			// Always succeed. We expect all data to be delivered as is.
			decisionFunc: func(ids idSet) error { return nil },
		},
		{
			name:         "random_non_permanent_error",
			decisionFunc: randomNonPermanentErrorConsumeDecision,
		},
		{
			name:         "random_permanent_error",
			decisionFunc: randomPermanentErrorConsumeDecision,
		},
		{
			name:         "random_error",
			decisionFunc: randomErrorsConsumeDecision,
		},
	}

	for _, scenario := range scenarios {
		params.T.Run(
			scenario.name, func(t *testing.T) {
				checkConsumeContractScenario(params, scenario.decisionFunc)
			},
		)
	}
}

func checkConsumeContractScenario(params CheckConsumeContractParams, decisionFunc func(ids idSet) error) {
	consumer := &mockConsumer{t: params.T, consumeDecisionFunc: decisionFunc}
	ctx := context.Background()

	// Create and start the receiver.
	var receiver component.Component
	var err error
	switch params.DataType {
	case component.DataTypeLogs:
		receiver, err = params.Factory.CreateLogsReceiver(ctx, NewNopCreateSettings(), params.Config, consumer)
	case component.DataTypeTraces:
		receiver, err = params.Factory.CreateTracesReceiver(ctx, NewNopCreateSettings(), params.Config, consumer)
	case component.DataTypeMetrics:
		// TODO: add metrics support to mockConsumer so that this case can be also implemented.
		require.FailNow(params.T, "DataTypeMetrics is not implemented")
	default:
		require.FailNow(params.T, "must specify a valid DataType to test for")
	}

	require.NoError(params.T, err)

	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(params.T, err)

	// Begin generating data to the receiver.

	var generatedIds idSet
	var generatedIndex int64
	var mux sync.Mutex
	var wg sync.WaitGroup

	const concurrency = 4

	params.Generator.Start()
	defer params.Generator.Stop()

	// Create concurrent goroutines that use the generator.
	// The total number of generator calls will be equal to params.GenerateCount.

	for j := 0; j < concurrency; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for atomic.AddInt64(&generatedIndex, 1) <= int64(params.GenerateCount) {
				ids := params.Generator.Generate()
				require.Greater(params.T, len(ids), 0)

				mux.Lock()
				duplicates := generatedIds.mergeSlice(ids)
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
		acceptedAndDropped, duplicates := consumer.acceptedAndDropped()
		if len(duplicates) != 0 {
			assert.Failf(params.T, "found duplicate elements in received and dropped data", "keys=%v", duplicates)
		}
		// Compare accepted+dropped with generated. Once they are equal it means all data is seen by the consumer.
		missingInOther, onlyInOther := generatedIds.compare(acceptedAndDropped)
		return len(missingInOther) == 0 && len(onlyInOther) == 0
	}, 5*time.Second, 10*time.Millisecond)

	// Do some final checks. Need the union of accepted and dropped data again.
	acceptedAndDropped, duplicates := consumer.acceptedAndDropped()
	if len(duplicates) != 0 {
		assert.Failf(params.T, "found duplicate elements in accepted and dropped data", "keys=%v", duplicates)
	}

	// Make sure generated and accepted+dropped are exactly the same.

	missingInOther, onlyInOther := generatedIds.compare(acceptedAndDropped)
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
		len(generatedIds),
		len(consumer.acceptedIds),
		len(consumer.droppedIds),
		consumer.nonPermanentFailures,
	)
}

// idSet is a set of unique ids of data elements used in the test (logs, spans or metric data points).
type idSet map[UniqueIDAttrVal]bool

// compare to another set and calculate the differences from this set.
func (ds idSet) compare(other idSet) (missingInOther, onlyInOther []UniqueIDAttrVal) {
	for k := range ds {
		if _, ok := other[k]; !ok {
			missingInOther = append(missingInOther, k)
		}
	}
	for k := range other {
		if _, ok := ds[k]; !ok {
			onlyInOther = append(onlyInOther, k)
		}
	}
	return
}

// merge another set into this one and return a list of duplicate ids.
func (ds *idSet) merge(other idSet) (duplicates []UniqueIDAttrVal) {
	if *ds == nil {
		*ds = map[UniqueIDAttrVal]bool{}
	}
	for k, v := range other {
		if _, ok := (*ds)[k]; ok {
			duplicates = append(duplicates, k)
		} else {
			(*ds)[k] = v
		}
	}
	return
}

// merge another set into this one and return a list of duplicate ids.
func (ds *idSet) mergeSlice(other []UniqueIDAttrVal) (duplicates []UniqueIDAttrVal) {
	if *ds == nil {
		*ds = map[UniqueIDAttrVal]bool{}
	}
	for _, id := range other {
		if _, ok := (*ds)[id]; ok {
			duplicates = append(duplicates, id)
		} else {
			(*ds)[id] = true
		}
	}
	return
}

// union computes the union of this and another sets. A new set if created to return the result.
// Also returns a list of any duplicate ids found.
func (ds *idSet) union(other idSet) (union idSet, duplicates []UniqueIDAttrVal) {
	union = map[UniqueIDAttrVal]bool{}
	for k, v := range *ds {
		union[k] = v
	}
	for k, v := range other {
		if _, ok := union[k]; ok {
			duplicates = append(duplicates, k)
		} else {
			union[k] = v
		}
	}
	return
}

// A function that returns a value indicating what the receiver's next consumer decides
// to do as a result of ConsumeLogs/Trace/Metrics call.
// The result of the decision function becomes the return value of ConsumeLogs/Trace/Metrics.
// Supplying different decision functions allows to test different scenarios of the contract
// between the receiver and it next consumer.
type consumeDecisionFunc func(ids idSet) error

var errNonPermanent = errors.New("non permanent error")
var errPermanent = errors.New("permanent error")

// randomNonPermanentErrorConsumeDecision is a decision function that succeeds approximately
// half of the time and fails with a non-permanent error the rest of the time.
func randomNonPermanentErrorConsumeDecision(_ idSet) error {
	if rand.Float32() < 0.5 {
		return errNonPermanent
	}
	return nil
}

// randomPermanentErrorConsumeDecision is a decision function that succeeds approximately
// half of the time and fails with a permanent error the rest of the time.
func randomPermanentErrorConsumeDecision(_ idSet) error {
	if rand.Float32() < 0.5 {
		return consumererror.NewPermanent(errPermanent)
	}
	return nil
}

// randomErrorsConsumeDecision is a decision function that succeeds approximately
// a third of the time, fails with a permanent error the third of the time and fails with
// a non-permanent error the rest of the time.
func randomErrorsConsumeDecision(_ idSet) error {
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
	acceptedIds          idSet
	droppedIds           idSet
	nonPermanentFailures int
}

func (m *mockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (m *mockConsumer) ConsumeTraces(_ context.Context, data ptrace.Traces) error {
	ids, err := idSetFromTraces(data)
	require.NoError(m.t, err)
	return m.consume(ids)
}

// idSetFromTraces computes an idSet from given ptrace.Traces. The idSet will contain ids of all spans.
func idSetFromTraces(data ptrace.Traces) (idSet, error) {
	ds := map[UniqueIDAttrVal]bool{}
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
				if key.Type() != pcommon.ValueTypeStr {
					return ds, fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
				}
				ds[UniqueIDAttrVal(key.Str())] = true
			}
		}
	}
	return ds, nil
}

func (m *mockConsumer) ConsumeLogs(_ context.Context, data plog.Logs) error {
	ids, err := idSetFromLogs(data)
	require.NoError(m.t, err)
	return m.consume(ids)
}

// idSetFromLogs computes an idSet from given plog.Logs. The idSet will contain ids of all log records.
func idSetFromLogs(data plog.Logs) (idSet, error) {
	ds := map[UniqueIDAttrVal]bool{}
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
				if key.Type() != pcommon.ValueTypeStr {
					return ds, fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
				}
				ds[UniqueIDAttrVal(key.Str())] = true
			}
		}
	}
	return ds, nil
}

// TODO: Implement mockConsumer.ConsumeMetrics()

// consume the elements with the specified ids, regardless of the element data type.
func (m *mockConsumer) consume(ids idSet) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Consult with user-defined decision function to decide what to do with the data.
	if err := m.consumeDecisionFunc(ids); err != nil {
		// The decision is to return an error to the receiver.

		if consumererror.IsPermanent(err) {
			// It is a permanent error, which means we need to drop the data.
			// Remember the ids of dropped elements.
			duplicates := m.droppedIds.merge(ids)
			require.Empty(m.t, duplicates, "elements that were dropped previously were sent again")
		} else {
			// It is a non-permanent error. Don't add it to the drop list. Remember the number of
			// failures to print at the end of the test.
			m.nonPermanentFailures++
		}
		// Return the error to the receiver.
		return err
	}

	// The decision is a success. Remember the ids of the data in the accepted list.
	duplicates := m.acceptedIds.merge(ids)
	require.Empty(m.t, duplicates, "elements that were accepted previously were sent again")
	return nil
}

// Calculate union of accepted and dropped ids.
// Returns the union and the list of duplicates between the two sets (if any)
func (m *mockConsumer) acceptedAndDropped() (acceptedAndDropped idSet, duplicates []UniqueIDAttrVal) {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.acceptedIds.union(m.droppedIds)
}

func CreateOneLogWithID(id UniqueIDAttrVal) plog.Logs {
	data := plog.NewLogs()
	data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().PutStr(
		UniqueIDAttrName,
		string(id),
	)
	return data
}
