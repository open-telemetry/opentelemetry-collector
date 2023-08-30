// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest // import "go.opentelemetry.io/collector/exporter/exportertest"

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// UniqueIDAttrName is the attribute name that is used in log records/spans/datapoints as the unique identifier.
const UniqueIDAttrName = "test_id"

// UniqueIDAttrVal is the value type of the UniqueIDAttrName.
type UniqueIDAttrVal string

type CheckConsumeContractParams struct {
	T *testing.T
	// Factory that allows to create an exporter.
	Factory exporter.Factory
	// DataType to test for.
	DataType component.DataType
	// Config of the exporter to use.
	Config               component.Config
	NumberOfTestElements int
	MockReceiver         MockReceiver
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
				checkConsumeContractScenario(params, scenario.decisionFunc, scenario.checkIfTestPassed)
			},
		)
	}
}

func checkConsumeContractScenario(params CheckConsumeContractParams, decisionFunc func() error, checkIfTestPassed func(*testing.T, int, requestCounter)) {

	switch params.DataType {
	case component.DataTypeLogs:
		checkLogs(params, decisionFunc, checkIfTestPassed)
	case component.DataTypeTraces:
		checkTraces(params, decisionFunc, checkIfTestPassed)
	case component.DataTypeMetrics:
		checkMetrics(params, decisionFunc, checkIfTestPassed)
	default:
		require.FailNow(params.T, "must specify a valid DataType to test for")
	}

}

func checkMetrics(params CheckConsumeContractParams, decisionFunc func() error, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	receiver := params.MockReceiver
	ctx := context.Background()

	var exp exporter.Metrics
	var err error
	exp, err = params.Factory.CreateMetricsExporter(ctx, NewNopCreateSettings(), params.Config)
	require.NoError(params.T, err)
	require.NotNil(params.T, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(params.T, err)

	defer func(exp exporter.Metrics, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(params.T, err)
		receiver.clearCounters()
	}(exp, ctx)

	receiver.setExportErrorFunction(decisionFunc)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := UniqueIDAttrVal(strconv.Itoa(i))
		fmt.Println("Preparing metric number: ", id)
		data := CreateOneMetricWithID(id)

		err = exp.ConsumeMetrics(ctx, data)
	}

	reqCounter := receiver.getReqCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)
	checkIfTestPassed(params.T, params.NumberOfTestElements, reqCounter)
}

func checkTraces(params CheckConsumeContractParams, decisionFunc func() error, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	receiver := params.MockReceiver
	ctx := context.Background()

	var exp exporter.Traces
	var err error
	exp, err = params.Factory.CreateTracesExporter(ctx, NewNopCreateSettings(), params.Config)
	require.NoError(params.T, err)
	require.NotNil(params.T, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(params.T, err)

	defer func(exp exporter.Traces, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(params.T, err)
		receiver.clearCounters()
	}(exp, ctx)

	receiver.setExportErrorFunction(decisionFunc)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := UniqueIDAttrVal(strconv.Itoa(i))
		fmt.Println("Preparing trace number: ", id)
		data := CreateOneTraceWithID(id)

		err = exp.ConsumeTraces(ctx, data)
	}

	reqCounter := receiver.getReqCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)
	checkIfTestPassed(params.T, params.NumberOfTestElements, reqCounter)
}

func checkLogs(params CheckConsumeContractParams, decisionFunc func() error, checkIfTestPassed func(*testing.T, int, requestCounter)) {
	receiver := params.MockReceiver
	ctx := context.Background()

	var exp exporter.Logs
	var err error
	exp, err = params.Factory.CreateLogsExporter(ctx, NewNopCreateSettings(), params.Config)
	require.NoError(params.T, err)
	require.NotNil(params.T, exp)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(params.T, err)

	defer func(exp exporter.Logs, ctx context.Context) {
		err = exp.Shutdown(ctx)
		require.NoError(params.T, err)
		receiver.clearCounters()
	}(exp, ctx)

	receiver.setExportErrorFunction(decisionFunc)

	for i := 0; i < params.NumberOfTestElements; i++ {
		id := UniqueIDAttrVal(strconv.Itoa(i))
		fmt.Println("Preparing log number: ", id)
		data := CreateOneLogWithID(id)

		err = exp.ConsumeLogs(ctx, data)
	}

	reqCounter := receiver.getReqCounter()
	// The overall number of requests sent by exporter
	fmt.Printf("Number of export tries: %d\n", reqCounter.total)
	// Successfully delivered items
	fmt.Printf("Total items received successfully: %d\n", reqCounter.success)
	// Number of errors that happened
	fmt.Printf("Number of permanent errors: %d\n", reqCounter.error.permanent)
	fmt.Printf("Number of non-permanent errors: %d\n", reqCounter.error.nonpermanent)
	checkIfTestPassed(params.T, params.NumberOfTestElements, reqCounter)
}

// Test is successful if all the elements were received successfully and no error was returned
func alwaysSucceedsPassed(t *testing.T, allRecordsNumber int, reqCounter requestCounter) {
	require.Equal(t, allRecordsNumber, reqCounter.success)
	require.Equal(t, reqCounter.total, allRecordsNumber)
	require.Equal(t, reqCounter.error.nonpermanent, 0)
	require.Equal(t, reqCounter.error.permanent, 0)
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

func CreateOneLogWithID(id UniqueIDAttrVal) plog.Logs {
	data := plog.NewLogs()
	data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().PutStr(
		UniqueIDAttrName,
		string(id),
	)
	return data
}

func CreateOneTraceWithID(id UniqueIDAttrVal) ptrace.Traces {
	data := ptrace.NewTraces()
	data.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().PutStr(
		UniqueIDAttrName,
		string(id),
	)
	return data
}

func CreateOneMetricWithID(id UniqueIDAttrVal) pmetric.Metrics {
	data := pmetric.NewMetrics()
	data.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram().DataPoints().AppendEmpty().Attributes().PutStr(UniqueIDAttrName, string(id))
	return data
}
