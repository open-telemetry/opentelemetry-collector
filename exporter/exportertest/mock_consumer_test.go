// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func createLog(id string) plog.Logs {
	validData := plog.NewLogs()
	validData.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().PutStr(
		uniqueIDAttrName,
		id,
	)
	return validData
}

func createTrace(id string) ptrace.Traces {
	validData := ptrace.NewTraces()
	validData.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().PutStr(
		uniqueIDAttrName,
		id,
	)
	return validData
}

func createMetric(id string) pmetric.Metrics {
	validData := pmetric.NewMetrics()
	validData.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram().DataPoints().AppendEmpty().Attributes().PutStr(uniqueIDAttrName, id)
	return validData
}

func TestIDFromMetrics(t *testing.T) {
	// Test case 1: Valid data
	id := "metric_id"
	validData := createMetric(id)
	metricID, err := idFromMetrics(validData)
	assert.Equal(t, metricID, id)
	require.NoError(t, err)

	// Test case 2: Missing uniqueIDAttrName attribute
	invalidData := pmetric.NewMetrics() // Create an invalid pmetric.Metrics object with missing attribute
	invalidData.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram().DataPoints().AppendEmpty().Attributes()
	_, err = idFromMetrics(invalidData)
	require.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is missing", uniqueIDAttrName))

	// Test case 3: Wrong attribute type
	var intID int64 = 12
	wrongAttribute := pmetric.NewMetrics() // Create a valid pmetric.Metrics object
	wrongAttribute.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().
		SetEmptyHistogram().DataPoints().AppendEmpty().Attributes().PutInt(uniqueIDAttrName, intID)
	_, err = idFromMetrics(wrongAttribute)
	assert.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is wrong type Int", uniqueIDAttrName))
}

func TestIDFromTraces(t *testing.T) {
	// Test case 1: Valid data
	id := "trace_id"
	validData := createTrace(id)
	traceID, err := idFromTraces(validData)
	assert.Equal(t, traceID, id)
	require.NoError(t, err)

	// Test case 2: Missing uniqueIDAttrName attribute
	invalidData := ptrace.NewTraces()
	invalidData.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes()
	_, err = idFromTraces(invalidData)
	require.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is missing", uniqueIDAttrName))

	// Test case 3: Wrong attribute type
	var intID int64 = 12
	wrongAttribute := ptrace.NewTraces()
	wrongAttribute.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().
		PutInt(uniqueIDAttrName, intID)
	_, err = idFromTraces(wrongAttribute)
	assert.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is wrong type Int", uniqueIDAttrName))
}

func TestIDFromLogs(t *testing.T) {
	// Test case 1: Valid data
	id := "log_id"
	validData := createLog(id)
	logID, err := idFromLogs(validData)
	assert.Equal(t, logID, id)
	require.NoError(t, err)

	// Test case 2: Missing uniqueIDAttrName attribute
	invalidData := plog.NewLogs()
	invalidData.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes()
	_, err = idFromLogs(invalidData)
	require.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is missing", uniqueIDAttrName))

	// Test case 3: Wrong attribute type
	var intID int64 = 12
	wrongAttribute := plog.NewLogs() // Create a valid plog.Metrics object
	wrongAttribute.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Attributes().
		PutInt(uniqueIDAttrName, intID)
	_, err = idFromLogs(wrongAttribute)
	assert.EqualError(t, err, fmt.Sprintf("invalid data element, attribute %q is wrong type Int", uniqueIDAttrName))
}

func returnNonPermanentError() error {
	return errNonPermanent
}

func returnPermanentError() error {
	return errPermanent
}

func TestConsumeLogsNonPermanent(t *testing.T) {
	mc := newMockConsumer(returnNonPermanentError)
	validData := createLog("logId")
	err := mc.ConsumeLogs(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 1, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeLogsPermanent(t *testing.T) {
	mc := newMockConsumer(returnPermanentError)
	validData := createLog("logId")
	err := mc.ConsumeLogs(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 1, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeLogsSuccess(t *testing.T) {
	mc := newMockConsumer(func() error { return nil })
	validData := createLog("logId")
	err := mc.ConsumeLogs(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 1, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeTracesNonPermanent(t *testing.T) {
	mc := newMockConsumer(returnNonPermanentError)
	validData := createTrace("traceId")
	err := mc.ConsumeTraces(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 1, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeTracesPermanent(t *testing.T) {
	mc := newMockConsumer(returnPermanentError)
	validData := createTrace("traceId")
	err := mc.ConsumeTraces(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 1, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeTracesSuccess(t *testing.T) {
	mc := newMockConsumer(func() error { return nil })
	validData := createTrace("traceId")
	err := mc.ConsumeTraces(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 1, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeMetricsNonPermanent(t *testing.T) {
	mc := newMockConsumer(returnNonPermanentError)
	validData := createMetric("metricId")
	err := mc.ConsumeMetrics(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 1, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeMetricsPermanent(t *testing.T) {
	mc := newMockConsumer(returnPermanentError)
	validData := createMetric("metricId")
	err := mc.ConsumeMetrics(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 1, mc.reqCounter.error.permanent)
	assert.Equal(t, 0, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestConsumeMetricsSuccess(t *testing.T) {
	mc := newMockConsumer(func() error { return nil })
	validData := createMetric("metricId")
	err := mc.ConsumeMetrics(context.Background(), validData)
	if err != nil {
		return
	}
	assert.Equal(t, 0, mc.reqCounter.error.nonpermanent)
	assert.Equal(t, 0, mc.reqCounter.error.permanent)
	assert.Equal(t, 1, mc.reqCounter.success)
	assert.Equal(t, 1, mc.reqCounter.total)
}

func TestCapabilities(t *testing.T) {
	mc := newMockConsumer(func() error { return nil })
	assert.Equal(t, consumer.Capabilities{}, mc.Capabilities())
}
