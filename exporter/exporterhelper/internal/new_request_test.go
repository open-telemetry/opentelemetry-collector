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
