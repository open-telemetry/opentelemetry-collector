// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest // import "go.opentelemetry.io/collector/exporter/exportertest"

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// uniqueIDAttrName is the attribute name that is used in log records/spans/datapoints as the unique identifier.
const uniqueIDAttrName = "test_id"

// uniqueIDAttrVal is the value type of the uniqueIDAttrName.
type uniqueIDAttrVal string

type CheckConsumeContractParams struct {
	T                    *testing.T
	NumberOfTestElements int
	Signal               pipeline.Signal
	// ExporterFactory to create an exporter to be tested.
	ExporterFactory exporter.Factory
	ExporterConfig  component.Config
	// ReceiverFactory to create a mock receiver.
	ReceiverFactory receiver.Factory
	ReceiverConfig  component.Config
}

func CheckConsumeContract(params CheckConsumeContractParams) {
	// Different scenarios to test for.
	// The decision function defines the testing scenario (i.e. to test for
	// success case or for error case or a mix of both). See for example randomErrorsConsumeDecision.
	scenarios := []struct {
		name              string
		decisionFunc      func() error
		checkIfTestPassed func(*testing.T, int, requestCounter)
	}{
		{
			name: "always_succeed",
			// Always succeed. We expect all data to be delivered as is.
			decisionFunc:      func() error { return nil },
			checkIfTestPassed: alwaysSucceedsPassed,
		},
		{
			name:              "random_non_permanent_error",
			decisionFunc:      randomNonPermanentErrorConsumeDecision,
			checkIfTestPassed: randomNonPermanentErrorConsumeDecisionPassed,
		},
		{
			name:              "random_permanent_error",
			decisionFunc:      randomPermanentErrorConsumeDecision,
			checkIfTestPassed: randomPermanentErrorConsumeDecisionPassed,
		},
		{
			name:              "random_error",
			decisionFunc:      randomErrorsConsumeDecision,
			checkIfTestPassed: randomErrorConsumeDecisionPassed,
		},
	}
	for _, scenario := range scenarios {
		params.T.Run(
			scenario.name, func(t *testing.T) {
				checkConsumeContractScenario(t, params, scenario.decisionFunc, scenario.checkIfTestPassed)
			},
		)
	}
}

func checkConsumeContractScenario(t *testing.T, params CheckConsumeContractParams, decisionFunc func() error, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	mockConsumerInstance := newMockConsumer(decisionFunc)
	switch params.Signal {
	case pipeline.SignalLogs:
		r, err := params.ReceiverFactory.CreateLogs(context.Background(), receivertest.NewNopSettings(params.ReceiverFactory.Type()), params.ReceiverConfig, &mockConsumerInstance)
		require.NoError(t, err)
		require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
		checkLogs(t, params, r, &mockConsumerInstance, checkIfTestPassed)
	case pipeline.SignalTraces:
		r, err := params.ReceiverFactory.CreateTraces(context.Background(), receivertest.NewNopSettings(params.ReceiverFactory.Type()), params.ReceiverConfig, &mockConsumerInstance)
		require.NoError(t, err)
		require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
		checkTraces(t, params, r, &mockConsumerInstance, checkIfTestPassed)
	case pipeline.SignalMetrics:
		r, err := params.ReceiverFactory.CreateMetrics(context.Background(), receivertest.NewNopSettings(params.ReceiverFactory.Type()), params.ReceiverConfig, &mockConsumerInstance)
		require.NoError(t, err)
		require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
		checkMetrics(t, params, r, &mockConsumerInstance, checkIfTestPassed)
	default:
		require.FailNow(t, "must specify a valid DataType to test for")
	}
}

func checkMetrics(t *testing.T, params CheckConsumeContractParams, mockReceiver component.Component,
	mockConsumer *mockConsumer, checkIfTestPassed func(*testing.T, int, requestCounter),
) {
	ctx := context.Background()
	var exp exporter.Metrics
	var err error
	exp, err = params.ExporterFactory.CreateMetrics(ctx, NewNopSettings(params.ExporterFactory.Type()), params.ExporterConfig)
	require.NoError(t, err)
	require.NotNil(t, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	defer func(exp exporter.Metrics, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(t, err)
		err = mockReceiver.Shutdown(ctx)
		require.NoError(t, err)
		mockConsumer.clear()
	}(exp, ctx)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := uniqueIDAttrVal(strconv.Itoa(i))
		data := createOneMetricWithID(id)

		err = exp.ConsumeMetrics(ctx, data)
	}

	reqCounter := mockConsumer.getRequestCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)

	assert.EventuallyWithT(t, func(*assert.CollectT) {
		checkIfTestPassed(t, params.NumberOfTestElements, *reqCounter)
	}, 2*time.Second, 100*time.Millisecond)
}

func checkTraces(t *testing.T, params CheckConsumeContractParams, mockReceiver component.Component, mockConsumer *mockConsumer, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	ctx := context.Background()
	var exp exporter.Traces
	var err error
	exp, err = params.ExporterFactory.CreateTraces(ctx, NewNopSettings(params.ExporterFactory.Type()), params.ExporterConfig)
	require.NoError(t, err)
	require.NotNil(t, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	defer func(exp exporter.Traces, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(t, err)
		err = mockReceiver.Shutdown(ctx)
		require.NoError(t, err)
		mockConsumer.clear()
	}(exp, ctx)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := uniqueIDAttrVal(strconv.Itoa(i))
		data := createOneTraceWithID(id)

		err = exp.ConsumeTraces(ctx, data)
	}

	reqCounter := mockConsumer.getRequestCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)

	assert.EventuallyWithT(t, func(*assert.CollectT) {
		checkIfTestPassed(t, params.NumberOfTestElements, *reqCounter)
	}, 2*time.Second, 100*time.Millisecond)
}

func checkLogs(t *testing.T, params CheckConsumeContractParams, mockReceiver component.Component, mockConsumer *mockConsumer, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	ctx := context.Background()
	var exp exporter.Logs
	var err error
	exp, err = params.ExporterFactory.CreateLogs(ctx, NewNopSettings(params.ExporterFactory.Type()), params.ExporterConfig)
	require.NoError(t, err)
	require.NotNil(t, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	defer func(exp exporter.Logs, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(t, err)
		err = mockReceiver.Shutdown(ctx)
		require.NoError(t, err)
		mockConsumer.clear()
	}(exp, ctx)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := uniqueIDAttrVal(strconv.Itoa(i))
		data := createOneLogWithID(id)

		err = exp.ConsumeLogs(ctx, data)
	}
	reqCounter := mockConsumer.getRequestCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)

	assert.EventuallyWithT(t, func(*assert.CollectT) {
		checkIfTestPassed(t, params.NumberOfTestElements, *reqCounter)
	}, 2*time.Second, 100*time.Millisecond)
}

// Test is successful if all the elements were received successfully and no error was returned
func alwaysSucceedsPassed(t *testing.T, allRecordsNumber int, reqCounter requestCounter) {
	require.Equal(t, allRecordsNumber, reqCounter.success)
	require.Equal(t, allRecordsNumber, reqCounter.total)
	require.Equal(t, 0, reqCounter.error.nonpermanent)
	require.Equal(t, 0, reqCounter.error.permanent)
}

// Test is successful if all the elements were retried on non-permanent errors
func randomNonPermanentErrorConsumeDecisionPassed(t *testing.T, allRecordsNumber int, reqCounter requestCounter) {
	// more or equal tries than successes
	require.GreaterOrEqual(t, reqCounter.total, reqCounter.success)
	// it is retried on every error
	require.Equal(t, reqCounter.total-reqCounter.error.nonpermanent, reqCounter.success)
	require.Equal(t, allRecordsNumber+reqCounter.error.nonpermanent, reqCounter.total)
}

// Test is successful if the calls are not retried on permanent errors
func randomPermanentErrorConsumeDecisionPassed(t *testing.T, allRecordsNumber int, reqCounter requestCounter) {
	require.Equal(t, allRecordsNumber-reqCounter.error.permanent, reqCounter.success)
	require.Equal(t, reqCounter.total, allRecordsNumber)
}

// Test is successful if the calls are not retried on permanent errors
func randomErrorConsumeDecisionPassed(t *testing.T, allRecordsNumber int, reqCounter requestCounter) {
	require.Equal(t, allRecordsNumber-reqCounter.error.permanent, reqCounter.success)
	require.Equal(t, reqCounter.total, allRecordsNumber+reqCounter.error.nonpermanent)
}

func createOneLogWithID(id uniqueIDAttrVal) plog.Logs {
	data := plog.NewLogs()
	data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().PutStr(
		uniqueIDAttrName,
		string(id),
	)
	return data
}

func createOneTraceWithID(id uniqueIDAttrVal) ptrace.Traces {
	data := ptrace.NewTraces()
	data.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().PutStr(
		uniqueIDAttrName,
		string(id),
	)
	return data
}

func createOneMetricWithID(id uniqueIDAttrVal) pmetric.Metrics {
	data := pmetric.NewMetrics()
	data.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram().
		DataPoints().AppendEmpty().Attributes().PutStr(uniqueIDAttrName, string(id))
	return data
}
