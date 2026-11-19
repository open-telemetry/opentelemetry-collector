// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/statusutil"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

func TestHTTPRetryAfter(t *testing.T) {
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

func TestWriteErrorBoundsStatusDetails(t *testing.T) {
	// A status nested in its own details, as the backend of an exporter could return it.
	nested := &spb.Status{Code: int32(codes.InvalidArgument), Message: "inner"}
	for range 2000 {
		a, err := anypb.New(nested)
		require.NoError(t, err)
		nested = &spb.Status{Code: int32(codes.InvalidArgument), Details: []*anypb.Any{a}}
	}
	nestedAny, err := anypb.New(nested)
	require.NoError(t, err)
	retryInfo, err := anypb.New(&errdetails.RetryInfo{RetryDelay: durationpb.New(5 * time.Second)})
	require.NoError(t, err)
	errorInfo, err := anypb.New(&errdetails.ErrorInfo{Reason: "my reason"})
	require.NoError(t, err)
	st := status.FromProto(&spb.Status{
		Code:    int32(codes.ResourceExhausted),
		Message: "rejected",
		Details: []*anypb.Any{nestedAny, retryInfo, errorInfo},
	})

	for _, enc := range []encoder{pbEncoder, jsEncoder} {
		t.Run(enc.contentType(), func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, enc, st.Err(), http.StatusInternalServerError)

			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			assert.Equal(t, "5", rec.Header().Get("Retry-After"))
			got := &spb.Status{}
			if enc == pbEncoder {
				require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), got))
			} else {
				require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), got))
			}
			assert.Equal(t, int32(codes.ResourceExhausted), got.GetCode())
			assert.Equal(t, "rejected", got.GetMessage())

			// The other details are kept, the nested one is cut at the depth limit.
			require.Len(t, got.GetDetails(), 3)
			assert.True(t, got.GetDetails()[1].MessageIs((*errdetails.RetryInfo)(nil)))
			assert.True(t, got.GetDetails()[2].MessageIs((*errdetails.ErrorInfo)(nil)))
			assert.True(t, got.GetDetails()[0].MessageIs((*spb.Status)(nil)))
			assert.Less(t, rec.Body.Len(), 64*1024)
		})
	}
}

func BenchmarkMarshalStatus(b *testing.B) {
	retryInfo, err := anypb.New(&errdetails.RetryInfo{RetryDelay: durationpb.New(5 * time.Second)})
	require.NoError(b, err)
	errorInfo, err := anypb.New(&errdetails.ErrorInfo{Reason: "my reason", Domain: "example.com", Metadata: map[string]string{"k": "v"}})
	require.NoError(b, err)
	nested := func(levels int) *anypb.Any {
		n := &spb.Status{Code: int32(codes.InvalidArgument), Message: "innermost"}
		for range levels {
			a, err := anypb.New(n)
			require.NoError(b, err)
			n = &spb.Status{Code: int32(codes.InvalidArgument), Details: []*anypb.Any{a}}
		}
		a, err := anypb.New(n)
		require.NoError(b, err)
		return a
	}
	mk := func(details ...*anypb.Any) *status.Status {
		return status.FromProto(&spb.Status{Code: int32(codes.ResourceExhausted), Message: "rejected", Details: details})
	}
	statuses := []struct {
		name string
		st   *status.Status
	}{
		{"retry_info", mk(retryInfo)},
		{"ordinary", mk(errorInfo, retryInfo)},
		{"nested_3", mk(nested(3), retryInfo)},
		// The deepest nesting that fits in the 64 KiB a backend can send.
		{"nested_max", mk(nested(2600), retryInfo)},
	}
	for _, enc := range []encoder{pbEncoder, jsEncoder} {
		for _, tt := range statuses {
			b.Run("unbounded/"+enc.contentType()+"/"+tt.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, err := enc.marshalStatus(tt.st.Proto())
					require.NoError(b, err)
				}
			})
			b.Run("bounded/"+enc.contentType()+"/"+tt.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, err := enc.marshalStatus(statusutil.BoundDetails(tt.st).Proto())
					require.NoError(b, err)
				}
			})
		}
	}
}
