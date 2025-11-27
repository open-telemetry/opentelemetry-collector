// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestLogsRequest_NilLogger(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromLogsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, le)
	require.Equal(t, errNilLogger, err)
}

func TestLogsRequest_NilLogsConverter(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, le)
	require.Equal(t, errNilLogsConverter, err)
}

func TestLogsRequest_NilPushLogsData(t *testing.T) {
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), requesttest.RequestFromLogsFunc(nil), nil)
	require.Nil(t, le)
	require.Equal(t, errNilConsumeRequest, err)
}

func TestLogsRequest_Default(t *testing.T) {
	ld := plog.NewLogs()
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, le.Capabilities())
	assert.NoError(t, le.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, le.Shutdown(context.Background()))
}

func TestLogsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithCapabilities(capabilities))
	require.NoError(t, err)
	require.NotNil(t, le)

	assert.Equal(t, capabilities, le.Capabilities())
}

func TestLogsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdown))
	assert.NotNil(t, le)
	assert.NoError(t, err)

	assert.NoError(t, le.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsRequest_Default_ConvertError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("convert_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(want), sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, consumererror.NewPermanent(want), le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_Default_ExportError(t *testing.T) {
	ld := plog.NewLogs()
	want := errors.New("export_error")
	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), sendertest.NewErrSenderFunc[request.Request](want))
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, want, le.ConsumeLogs(context.Background(), ld))
}

func TestLogsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	le, err := NewLogsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromLogsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdownErr))
	assert.NotNil(t, le)
	require.NoError(t, err)

	assert.Equal(t, want, le.Shutdown(context.Background()))
}

func TestTracesRequest_NilLogger(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromTracesFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, te)
	require.Equal(t, errNilLogger, err)
}

func TestTracesRequest_NilTracesConverter(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, te)
	require.Equal(t, errNilTracesConverter, err)
}

func TestTracesRequest_NilPushTraceData(t *testing.T) {
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), requesttest.RequestFromTracesFunc(nil), nil)
	require.Nil(t, te)
	require.Equal(t, errNilConsumeRequest, err)
}

func TestTracesRequest_Default(t *testing.T) {
	td := ptrace.NewTraces()
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, te.Capabilities())
	assert.NoError(t, te.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), td))
	assert.NoError(t, te.Shutdown(context.Background()))
}

func TestTracesRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithCapabilities(capabilities))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, capabilities, te.Capabilities())
}

func TestTracesRequest_Default_ConvertError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("convert_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(want), sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)
	require.NotNil(t, te)
	require.Equal(t, consumererror.NewPermanent(want), te.ConsumeTraces(context.Background(), td))
}

func TestTracesRequest_Default_ExportError(t *testing.T) {
	td := ptrace.NewTraces()
	want := errors.New("export_error")
	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), sendertest.NewErrSenderFunc[request.Request](want))
	require.NoError(t, err)
	require.NotNil(t, te)
	require.Equal(t, want, te.ConsumeTraces(context.Background(), td))
}

func TestTracesRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.NoError(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTracesRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTracesRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromTracesFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	require.NoError(t, err)

	assert.Equal(t, want, te.Shutdown(context.Background()))
}

func TestMetricsRequest_NilLogger(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exporter.Settings{}, requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetricsRequest_NilMetricsConverter(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), nil, sendertest.NewNopSenderFunc[request.Request]())
	require.Nil(t, me)
	require.Equal(t, errNilMetricsConverter, err)
}

func TestMetricsRequest_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType), requesttest.RequestFromMetricsFunc(nil), nil)
	require.Nil(t, me)
	require.Equal(t, errNilConsumeRequest, err)
}

func TestMetricsRequest_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsRequest_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithCapabilities(capabilities))
	require.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetricsRequest_Default_ConvertError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("convert_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(want), sendertest.NewNopSenderFunc[request.Request]())
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, consumererror.NewPermanent(want), me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_Default_ExportError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("export_error")
	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewErrSenderFunc[request.Request](want))
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRequest_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsRequest_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsRequest(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		requesttest.RequestFromMetricsFunc(nil), sendertest.NewNopSenderFunc[request.Request](), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}
