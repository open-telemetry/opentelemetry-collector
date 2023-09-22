// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package exportertest // import "go.opentelemetry.io/collector/exporter/exportertest"

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var errNonPermanent = status.Error(codes.DeadlineExceeded, "non Permanent error")
var errPermanent = status.Error(codes.Internal, "Permanent error")

type MockReceiver interface {
	Start() error
	Stop() error
	RequestCounter() RequestCounter
}

type MockReceiverFactory func(consumer MockConsumer) MockReceiver

// // randomNonPermanentErrorConsumeDecision is a decision function that succeeds approximately
// // half of the time and fails with a non-permanent error the rest of the time.
func randomNonPermanentErrorConsumeDecision() error {
	if rand.Float32() < 0.5 {
		return errNonPermanent
	}
	return nil
}

// randomPermanentErrorConsumeDecision is a decision function that succeeds approximately
// half of the time and fails with a permanent error the rest of the time.
func randomPermanentErrorConsumeDecision() error {
	if rand.Float32() < 0.5 {
		return consumererror.NewPermanent(errPermanent)
	}
	return nil
}

// randomErrorsConsumeDecision is a decision function that succeeds approximately
// a third of the time, fails with a permanent error the third of the time and fails with
// a non-permanent error the rest of the time.
func randomErrorsConsumeDecision() error {
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

type MockConsumer interface {
	ConsumeLogs(_ context.Context, ld plog.Logs) error
	ConsumeMetrics(_ context.Context, md pmetric.Metrics) error
	ConsumeTraces(_ context.Context, td ptrace.Traces) error
	Capabilities() consumer.Capabilities
	Clear()
	RequestCounter() RequestCounter
}

type mockConsumer struct {
	reqCounter          RequestCounter
	mux                 sync.Mutex
	exportErrorFunction func() error
	ReceivedTraces      []ptrace.Traces
	ReceivedMetrics     []pmetric.Metrics
	ReceivedLogs        []plog.Logs
}

func NewMockConsumer(decisionFunc func() error) MockConsumer {
	return &mockConsumer{
		reqCounter:          newRequestCounter(),
		mux:                 sync.Mutex{},
		exportErrorFunction: decisionFunc,
		ReceivedTraces:      nil,
		ReceivedMetrics:     nil,
		ReceivedLogs:        nil,
	}
}

type RequestCounter struct {
	success int
	error   errorCounter
	total   int
}

type errorCounter struct {
	permanent    int
	nonpermanent int
}

func newErrorCounter() errorCounter {
	return errorCounter{
		permanent:    0,
		nonpermanent: 0,
	}
}

func newRequestCounter() RequestCounter {
	return RequestCounter{
		success: 0,
		error:   newErrorCounter(),
		total:   0,
	}
}

func (r *mockConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	logID, _ := idFromLogs(ld)
	if generatedError != nil {
		r.processError(generatedError, "log", logID)
		return generatedError
	}
	fmt.Println("Successfully sent log number:", logID)
	r.reqCounter.success++
	r.ReceivedLogs = append(r.ReceivedLogs, ld)
	return nil
}

func (r *mockConsumer) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	traceID, _ := idFromTraces(td)
	if generatedError != nil {
		r.processError(generatedError, "log", traceID)
		return generatedError
	}
	fmt.Println("Successfully sent log number:", traceID)
	r.reqCounter.success++
	r.ReceivedTraces = append(r.ReceivedTraces, td)
	return nil
}

func (r *mockConsumer) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	traceID, _ := idFromMetrics(md)
	if generatedError != nil {
		r.processError(generatedError, "log", traceID)
		return generatedError
	}
	fmt.Println("Successfully sent log number:", traceID)
	r.reqCounter.success++
	r.ReceivedMetrics = append(r.ReceivedMetrics, md)
	return nil
}

func (r *mockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (r *mockConsumer) processError(err error, dataType string, idOfElement string) {
	if consumererror.IsPermanent(err) {
		fmt.Println("permanent error happened")
		fmt.Printf("Dropping %s number: %s\n", dataType, idOfElement)
		r.reqCounter.error.permanent++
	} else {
		fmt.Println("non-permanent error happened")
		fmt.Printf("Retrying %s number: %s\n", dataType, idOfElement)
		r.reqCounter.error.nonpermanent++
	}
}

func (r *mockConsumer) Clear() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter = newRequestCounter()
}

func (r *mockConsumer) RequestCounter() RequestCounter {
	return r.reqCounter
}

func idFromLogs(data plog.Logs) (string, error) {
	var logID string
	rss := data.ResourceLogs()
	key, exists := rss.At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().Get(UniqueIDAttrName)
	if !exists {
		return "", fmt.Errorf("invalid data element, attribute %q is missing", UniqueIDAttrName)
	}
	if key.Type() != pcommon.ValueTypeStr {
		return "", fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
	}
	logID = key.Str()
	return logID, nil
}

func idFromTraces(data ptrace.Traces) (string, error) {
	var traceID string
	rss := data.ResourceSpans()
	key, exists := rss.At(0).ScopeSpans().At(0).Spans().At(0).Attributes().Get(UniqueIDAttrName)
	if !exists {
		return "", fmt.Errorf("invalid data element, attribute %q is missing", UniqueIDAttrName)
	}
	if key.Type() != pcommon.ValueTypeStr {
		return "", fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
	}
	traceID = key.Str()
	return traceID, nil
}

func idFromMetrics(data pmetric.Metrics) (string, error) {
	var metricID string
	rss := data.ResourceMetrics()
	key, exists := rss.At(0).ScopeMetrics().At(0).Metrics().At(0).Histogram().DataPoints().At(0).Attributes().Get(UniqueIDAttrName)
	if !exists {
		return "", fmt.Errorf("invalid data element, attribute %q is missing", UniqueIDAttrName)
	}
	if key.Type() != pcommon.ValueTypeStr {
		return "", fmt.Errorf("invalid data element, attribute %q is wrong type %v", UniqueIDAttrName, key.Type())
	}
	metricID = key.Str()
	return metricID, nil
}
