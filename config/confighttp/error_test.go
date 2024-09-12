// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

func TestError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{
			name:   "permanent-error",
			err:    consumererror.NewPermanent(errors.New("Permanent error")),
			status: http.StatusBadRequest,
		},
		{
			name:   "non-permanent-error",
			err:    errors.New("Non permanent generic error"),
			status: http.StatusServiceUnavailable,
		},
		{
			name:   "non-permanent-too-many-requests",
			err:    memorylimiter.ErrDataRefused,
			status: http.StatusTooManyRequests,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hss := NewDefaultServerConfig()
			hss.Endpoint = "localhost:0"
			srv, err := hss.ToServer(
				context.Background(),
				componenttest.NewNopHost(),
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					Error(w, tt.err)
				}),
			)
			require.NoError(t, err)
			response := &httptest.ResponseRecorder{
				Body: bytes.NewBuffer(make([]byte, 0, 100)),
			}
			req, err := http.NewRequest(http.MethodGet, srv.Addr, bytes.NewBuffer([]byte{}))
			require.NoError(t, err, "Error creating request: %v", err)

			srv.Handler.ServeHTTP(response, req)
			require.Contains(t, string(response.Body.Bytes()), tt.err.Error())
		})
	}
}
