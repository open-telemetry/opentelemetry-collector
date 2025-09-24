// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

func TestHttpRetryAfter(t *testing.T) {
	tests := []struct {
		name                  string
		contentType           string
		err                   error
		expectedStatusCode    int
		expectedHasRetryAfter bool
		expectedRetryAfter    string
	}{
		{
			name:               "StatusErrorRetryableNoRetryAfter",
			err:                status.New(codes.DeadlineExceeded, "").Err(),
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			name: "StatusErrorRetryableWithZeroRetryAfter",
			err: func() error {
				st := status.New(codes.ResourceExhausted, "")
				dt, err := st.WithDetails(&errdetails.RetryInfo{
					RetryDelay: durationpb.New(0),
				})
				require.NoError(t, err)
				return dt.Err()
			}(),
			expectedStatusCode:    http.StatusTooManyRequests,
			expectedHasRetryAfter: true,
			expectedRetryAfter:    "0",
		},
		{
			name: "StatusErrorRetryableRetryAfter",
			err: func() error {
				st := status.New(codes.ResourceExhausted, "")
				dt, err := st.WithDetails(&errdetails.RetryInfo{
					RetryDelay: durationpb.New(13 * time.Second),
				})
				require.NoError(t, err)
				return dt.Err()
			}(),
			expectedStatusCode:    http.StatusTooManyRequests,
			expectedHasRetryAfter: true,
			expectedRetryAfter:    "13",
		},
		{
			name: "StatusErrorNotRetryableRetryAfter",
			err: func() error {
				st := status.New(codes.Unknown, "")
				dt, err := st.WithDetails(&errdetails.RetryInfo{
					RetryDelay: durationpb.New(12 * time.Second),
				})
				require.NoError(t, err)
				return dt.Err()
			}(),
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "StatusErrorNotRetryableNoRetryAfter",
			err:                status.New(codes.InvalidArgument, "").Err(),
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := newErrOrSinkConsumer()
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, sink)

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink.Reset()
			sink.SetConsumeError(tt.err)

			for _, dr := range generateDataRequests(t) {
				url := "http://" + addr + dr.path
				req := createHTTPRequest(t, url, "", "application/x-protobuf", dr.protoBytes)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)

				respBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				require.NoError(t, resp.Body.Close())
				// For cases like "application/json; charset=utf-8", the response will be only "application/json"
				require.True(t, strings.HasPrefix(strings.ToLower("application/x-protobuf"), resp.Header.Get("Content-Type")))
				if tt.expectedHasRetryAfter {
					require.Equal(t, tt.expectedRetryAfter, resp.Header.Get("Retry-After"))
				} else {
					require.Empty(t, resp.Header.Get("Retry-After"))
				}

				assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

				if tt.err == nil {
					tr := ptraceotlp.NewExportResponse()
					require.NoError(t, tr.UnmarshalProto(respBytes))
					sink.checkData(t, dr.data, 1)
				} else {
					errStatus := &spb.Status{}
					require.NoError(t, proto.Unmarshal(respBytes, errStatus))
					// The HTTP receiver transforms errors through GetStatusFromError
					// We need to get the expected transformed error, not the original
					expectedErr := errors.GetStatusFromError(tt.err)
					s, ok := status.FromError(expectedErr)
					require.True(t, ok)
					assert.True(t, proto.Equal(errStatus, s.Proto()))
				}
			}
		})
	}
}
