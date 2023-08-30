// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package exportertest // import "go.opentelemetry.io/collector/exporter/exportertest"

import (
	"context"
	"fmt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"math/rand"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

var errNonPermanent = status.Error(codes.DeadlineExceeded, "non Permanent error")
var errPermanent = status.Error(codes.Internal, "Permanent error")

type DecisionFunc func() error

type MockReceiver interface {
	clearCounters()
	setExportErrorFunction(decisionFunction func() error)
	srvStop()
	getReqCounter() requestCounter
}

// // randomNonPermanentErrorConsumeDecision is a decision function that succeeds approximately
// // half of the time and fails with a non-permanent error the rest of the time.
func randomNonPermanentErrorConsumeDecision() error {
	if rand.Float32() < 0.5 {
		return errNonPermanent
	}
	return nil
}

func CreateMockReceiver(dataType component.DataType, ln net.Listener) MockReceiver {
	switch dataType {
	case component.DataTypeLogs:
		return otlpLogsReceiverOnGRPCServer(ln)
	case component.DataTypeTraces:
		return otlpTracesReceiverOnGRPCServer(ln)
	case component.DataTypeMetrics:
		return otlpMetricsReceiverOnGRPCServer(ln)
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

type mockReceiver struct {
	srv                 *grpc.Server
	reqCounter          requestCounter
	mux                 sync.Mutex
	exportErrorFunction func() error
}

type requestCounter struct {
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

func newRequestCounter() requestCounter {
	return requestCounter{
		success: 0,
		error:   newErrorCounter(),
		total:   0,
	}
}

type mockMetricsReceiver struct {
	pmetricotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() pmetricotlp.ExportResponse
	lastRequest    pmetric.Metrics
}

type mockLogsReceiver struct {
	plogotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() plogotlp.ExportResponse
	lastRequest    plog.Logs
}

func (r *mockReceiver) srvStop() {
	r.srv.GracefulStop()
}

type mockTracesReceiver struct {
	ptraceotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() ptraceotlp.ExportResponse
	lastRequest    ptrace.Traces
}

func (r *mockReceiver) setExportErrorFunction(decisionFunction func() error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportErrorFunction = decisionFunction
}

func (r *mockLogsReceiver) Export(_ context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	logID, _ := idFromLogs(req.Logs())
	if generatedError != nil {
		r.processError(generatedError, "log", logID)
		return r.exportResponse(), generatedError
	}
	fmt.Println("Successfully sent log number:", logID)
	ld := req.Logs()
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.success++
	r.lastRequest = ld
	return r.exportResponse(), nil
}

func (r *mockTracesReceiver) Export(_ context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	traceID, _ := idFromTraces(req.Traces())
	if generatedError != nil {
		r.processError(generatedError, "trace", traceID)
		return r.exportResponse(), generatedError
	}
	fmt.Println("Successfully sent trace number:", traceID)
	td := req.Traces()
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.success++
	r.lastRequest = td
	return r.exportResponse(), nil
}

func (r *mockMetricsReceiver) Export(_ context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	r.reqCounter.total++
	generatedError := r.exportErrorFunction()
	metricID, _ := idFromMetrics(req.Metrics())
	if generatedError != nil {
		r.processError(generatedError, "metric", metricID)
		return r.exportResponse(), generatedError
	}
	fmt.Println("Successfully sent metric number:", metricID)
	md := req.Metrics()
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter.success++
	r.lastRequest = md
	return r.exportResponse(), nil
}

func (r *mockMetricsReceiver) createExporter(ctx context.Context, factory exporter.Factory, cfg component.Config) (exporter.Metrics, error) {
	exp, err := factory.CreateMetricsExporter(ctx, NewNopCreateSettings(), cfg)
	return exp, err
}

func (r *mockTracesReceiver) createExporter(ctx context.Context, factory exporter.Factory, cfg component.Config) (exporter.Traces, error) {
	exp, err := factory.CreateTracesExporter(ctx, NewNopCreateSettings(), cfg)
	return exp, err
}

func (r *mockLogsReceiver) createExporter(ctx context.Context, factory exporter.Factory, cfg component.Config) (exporter.Logs, error) {
	exp, err := factory.CreateLogsExporter(ctx, NewNopCreateSettings(), cfg)
	return exp, err
}

func (r *mockReceiver) processError(err error, dataType string, idOfElement string) {
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

func (r *mockReceiver) clearCounters() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.reqCounter = newRequestCounter()
}

func (r *mockReceiver) getReqCounter() requestCounter {
	return r.reqCounter
}

func otlpMetricsReceiverOnGRPCServer(ln net.Listener) *mockMetricsReceiver {
	rcv := &mockMetricsReceiver{
		mockReceiver: mockReceiver{
			srv:        grpc.NewServer(),
			reqCounter: newRequestCounter(),
		},
		exportResponse: pmetricotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	pmetricotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{
		mockReceiver: mockReceiver{
			srv:        grpc.NewServer(),
			reqCounter: newRequestCounter(),
		},
		exportResponse: plogotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	plogotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

func otlpTracesReceiverOnGRPCServer(ln net.Listener) *mockTracesReceiver {
	sopts := []grpc.ServerOption{}

	rcv := &mockTracesReceiver{
		mockReceiver: mockReceiver{
			srv:        grpc.NewServer(sopts...),
			reqCounter: newRequestCounter(),
		},
		exportResponse: ptraceotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	ptraceotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
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
