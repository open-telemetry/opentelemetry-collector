// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

func TestShutdownSender(t *testing.T) {
	errRetryable := errors.New("connection refused")
	errPermanent := consumererror.NewPermanent(errors.New("invalid data"))

	tests := []struct {
		name           string
		sendErr        error
		afterShutdown  bool
		wantShutdownEr bool
	}{
		{
			name:    "no_error_before_shutdown",
			sendErr: nil,
		},
		{
			name:          "no_error_after_shutdown",
			sendErr:       nil,
			afterShutdown: true,
		},
		{
			name:    "retryable_error_before_shutdown",
			sendErr: errRetryable,
		},
		{
			name:           "retryable_error_after_shutdown",
			sendErr:        errRetryable,
			afterShutdown:  true,
			wantShutdownEr: true,
		},
		{
			name:          "permanent_error_after_shutdown",
			sendErr:       errPermanent,
			afterShutdown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := newShutdownSender[request.Request](
				sender.NewSender(func(context.Context, request.Request) error { return tt.sendErr }))

			if tt.afterShutdown {
				require.NoError(t, ss.Shutdown(context.Background()))
			}

			err := ss.Send(context.Background(), &requesttest.FakeRequest{Items: 2})
			if tt.sendErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.sendErr)
			assert.Equal(t, tt.wantShutdownEr, experr.IsShutdownErr(err))
		})
	}
}
