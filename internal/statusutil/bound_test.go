// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statusutil

import (
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
)

// nestedStatus returns a status that contains itself, as a detail, levels times.
func nestedStatus(tb testing.TB, levels int) *spb.Status {
	tb.Helper()
	nested := &spb.Status{Code: int32(codes.InvalidArgument), Message: "innermost"}
	for range levels {
		a, err := anypb.New(nested)
		require.NoError(tb, err)
		nested = &spb.Status{Code: int32(codes.InvalidArgument), Details: []*anypb.Any{a}}
	}
	return nested
}

// depthOf returns how many levels of Status detail are nested in p.
func depthOf(tb testing.TB, p *spb.Status) int {
	tb.Helper()
	depth := 0
	for len(p.GetDetails()) > 0 {
		d := p.GetDetails()[0]
		if len(d.GetValue()) == 0 {
			break
		}
		depth++
		next := &spb.Status{}
		require.NoError(tb, d.UnmarshalTo(next))
		p = next
	}
	return depth
}

func TestBoundDetailsKeepsOrdinaryDetails(t *testing.T) {
	retryInfo, err := anypb.New(&errdetails.RetryInfo{RetryDelay: durationpb.New(5 * time.Second)})
	require.NoError(t, err)
	errorInfo, err := anypb.New(&errdetails.ErrorInfo{Reason: "my reason", Metadata: map[string]string{"k": "v"}})
	require.NoError(t, err)
	shallow, err := anypb.New(nestedStatus(t, 3))
	require.NoError(t, err)

	st := status.FromProto(&spb.Status{
		Code:    int32(codes.ResourceExhausted),
		Message: "rejected",
		Details: []*anypb.Any{retryInfo, errorInfo, shallow},
	})
	assert.Same(t, st, BoundDetails(st), "a status within the limits is returned as is")

	noDetails := status.New(codes.Internal, "no details")
	assert.Same(t, noDetails, BoundDetails(noDetails), "no details")
}

func TestBoundDetailsLimitsNesting(t *testing.T) {
	retryInfo, err := anypb.New(&errdetails.RetryInfo{RetryDelay: durationpb.New(5 * time.Second)})
	require.NoError(t, err)
	errorInfo, err := anypb.New(&errdetails.ErrorInfo{Reason: "my reason"})
	require.NoError(t, err)
	deep, err := anypb.New(nestedStatus(t, 3000))
	require.NoError(t, err)
	require.LessOrEqual(t, len(deep.GetValue()), MaxDetailBytes*10)

	in := status.FromProto(&spb.Status{
		Code:    int32(codes.ResourceExhausted),
		Message: "rejected",
		Details: []*anypb.Any{deep, retryInfo, errorInfo},
	})
	original := proto.Clone(in.Proto())

	out := BoundDetails(in)
	assert.Equal(t, int32(codes.ResourceExhausted), out.Proto().GetCode())
	assert.Equal(t, "rejected", out.Message())
	require.Len(t, out.Proto().GetDetails(), 3)

	// The deep detail is kept down to the maximum depth, its type is kept below it.
	assert.LessOrEqual(t, depthOf(t, &spb.Status{Details: out.Proto().GetDetails()[:1]}), MaxDetailDepth)
	// The other details are untouched.
	assert.True(t, proto.Equal(retryInfo, out.Proto().GetDetails()[1]))
	assert.True(t, proto.Equal(errorInfo, out.Proto().GetDetails()[2]))
	assert.NotNil(t, GetRetryInfo(out))

	// The input is not modified.
	assert.True(t, proto.Equal(original, in.Proto()))

	// The result can be expanded by protojson without the cost of the input.
	_, err = protojson.Marshal(out.Proto())
	require.NoError(t, err)
}

func TestBoundDetailsLimitsSize(t *testing.T) {
	big, err := anypb.New(&errdetails.DebugInfo{Detail: strings.Repeat("x", 2*MaxDetailBytes)})
	require.NoError(t, err)
	small, err := anypb.New(&errdetails.DebugInfo{Detail: "small"})
	require.NoError(t, err)

	out := BoundDetails(status.FromProto(&spb.Status{Code: int32(codes.Internal), Details: []*anypb.Any{big, small}}))
	require.Len(t, out.Proto().GetDetails(), 2)
	assert.Equal(t, big.GetTypeUrl(), out.Proto().GetDetails()[0].GetTypeUrl())
	assert.Empty(t, out.Proto().GetDetails()[0].GetValue())
	assert.True(t, proto.Equal(small, out.Proto().GetDetails()[1]))
}

func TestBoundDetailsLimitsCount(t *testing.T) {
	// A Status can hold nested details, so it is decoded and counted against the budget.
	info, err := anypb.New(nestedStatus(t, 0))
	require.NoError(t, err)
	details := make([]*anypb.Any, 0, 2*MaxDecodedDetails)
	for range 2 * MaxDecodedDetails {
		details = append(details, info)
	}
	out := BoundDetails(status.FromProto(&spb.Status{Code: int32(codes.Internal), Details: details}))
	require.Len(t, out.Proto().GetDetails(), len(details))
	kept := 0
	for _, d := range out.Proto().GetDetails() {
		if len(d.GetValue()) > 0 {
			kept++
		}
	}
	assert.Equal(t, MaxDecodedDetails, kept)
}

// benchStatuses returns the statuses used by the benchmarks: the common ones, and
// the largest ones a backend can send within the limits of the exporters.
func benchStatuses(tb testing.TB) map[string]*status.Status {
	tb.Helper()
	retryInfo, err := anypb.New(&errdetails.RetryInfo{RetryDelay: durationpb.New(5 * time.Second)})
	require.NoError(tb, err)
	errorInfo, err := anypb.New(&errdetails.ErrorInfo{Reason: "my reason", Domain: "example.com", Metadata: map[string]string{"k": "v"}})
	require.NoError(tb, err)
	debugInfo, err := anypb.New(&errdetails.DebugInfo{Detail: strings.Repeat("x", 512)})
	require.NoError(tb, err)
	nested3, err := anypb.New(nestedStatus(tb, 3))
	require.NoError(tb, err)
	// The deepest nesting that fits in the 64 KiB a backend can send.
	nestedMax, err := anypb.New(nestedStatus(tb, 2600))
	require.NoError(tb, err)
	badRequest := &errdetails.BadRequest{}
	for range 30000 {
		badRequest.FieldViolations = append(badRequest.FieldViolations, &errdetails.BadRequest_FieldViolation{})
	}
	bigBadRequest, err := anypb.New(badRequest)
	require.NoError(tb, err)

	mk := func(details ...*anypb.Any) *status.Status {
		return status.FromProto(&spb.Status{Code: int32(codes.ResourceExhausted), Message: "rejected", Details: details})
	}
	return map[string]*status.Status{
		"no_details":      mk(),
		"retry_info":      mk(retryInfo),
		"ordinary":        mk(errorInfo, debugInfo, retryInfo),
		"nested_3":        mk(nested3, retryInfo),
		"nested_max":      mk(nestedMax, retryInfo),
		"big_bad_request": mk(bigBadRequest, retryInfo),
	}
}

// getRetryInfoDecodingAll is GetRetryInfo as it was: it decodes every detail.
func getRetryInfoDecodingAll(s *status.Status) *errdetails.RetryInfo {
	for _, detail := range s.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			return t
		}
	}
	return nil
}

func BenchmarkGetRetryInfo(b *testing.B) {
	for name, st := range benchStatuses(b) {
		b.Run("old/"+name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = getRetryInfoDecodingAll(st)
			}
		})
		b.Run("new/"+name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = GetRetryInfo(st)
			}
		})
	}
}

func BenchmarkBoundDetails(b *testing.B) {
	for name, st := range benchStatuses(b) {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = BoundDetails(st)
			}
		})
	}
}
