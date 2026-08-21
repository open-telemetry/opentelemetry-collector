// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelpertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestNewNop(t *testing.T) {
	metrics := NewNop()
	value := exporterhelper.NewInt64Value(func() int64 { return 1 })

	require.NoError(t, metrics.RegisterQueueSize(value))
	require.NoError(t, metrics.RegisterQueueCapacity(value))
	metrics.RecordEnqueueFailure(context.Background(), 1)
	metrics.RecordEnqueueSize(context.Background(), 1, value)
	metrics.RecordBatchSendSize(context.Background(), 1, value)
	metrics.RecordInFlight(context.Background(), 1)
	metrics.RecordSent(context.Background(), 1)
	metrics.RecordSendFailure(context.Background(), 1)
	metrics.Shutdown()
}

func TestNewErr(t *testing.T) {
	errTest := errors.New("test")
	metrics := NewErr(errTest)
	value := exporterhelper.NewInt64Value(nil)

	require.ErrorIs(t, metrics.RegisterQueueSize(value), errTest)
	require.ErrorIs(t, metrics.RegisterQueueCapacity(value), errTest)
}
