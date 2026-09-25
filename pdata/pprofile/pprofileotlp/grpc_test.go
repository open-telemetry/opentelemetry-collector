// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeProfilesServer{t: t})
	wg := sync.WaitGroup{}
	wg.Go(func() {
		assert.NoError(t, s.Serve(lis))
	})
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	resolver.SetDefaultScheme("passthrough")
	cc, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewGRPCClient(cc)

	resp, err := logClient.Export(context.Background(), generateProfilesRequest())
	require.NoError(t, err)
	assert.Equal(t, NewExportResponse(), resp)
}

func TestGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeProfilesServer{t: t, err: errors.New("my error")})
	wg := sync.WaitGroup{}
	wg.Go(func() {
		assert.NoError(t, s.Serve(lis))
	})
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	cc, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewGRPCClient(cc)
	resp, err := logClient.Export(context.Background(), generateProfilesRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, ExportResponse{}, resp)
}

func TestGRPCExportDoesNotMutateInput(t *testing.T) {
	tests := []struct {
		name      string
		readOnly  bool
		serverErr error
	}{
		{name: "mutable/success"},
		{name: "mutable/error", serverErr: errors.New("my error")},
		{name: "read-only/success", readOnly: true},
		{name: "read-only/error", readOnly: true, serverErr: errors.New("my error")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			RegisterGRPCServer(s, &capturingProfilesServer{err: tc.serverErr})
			wg := sync.WaitGroup{}
			wg.Go(func() {
				assert.NoError(t, s.Serve(lis))
			})
			t.Cleanup(func() {
				s.Stop()
				wg.Wait()
			})

			cc, err := grpc.NewClient("passthrough:///bufnet",
				grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return lis.Dial()
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, cc.Close())
			})

			profiles := pprofile.NewProfiles()
			resourceProfiles := profiles.ResourceProfiles().AppendEmpty()
			resourceProfiles.Resource().Attributes().PutStr("service.name", "checkout")
			scopeProfiles := resourceProfiles.ScopeProfiles().AppendEmpty()
			scopeProfiles.Scope().Attributes().PutStr("scope.attr", "scope-value")

			want := pprofile.NewProfiles()
			profiles.CopyTo(want)
			if tc.readOnly {
				profiles.MarkReadOnly()
				want.MarkReadOnly()
			}

			_, err = NewGRPCClient(cc).Export(context.Background(), NewExportRequestFromProfiles(profiles))
			if tc.serverErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, want, profiles)
		})
	}
}

func TestGRPCExportUsesProfilesDictionary(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	var wireRequest *internal.ExportProfilesServiceRequest
	s := grpc.NewServer(grpc.UnaryInterceptor(func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		wireRequest = internal.CopyExportProfilesServiceRequest(nil, req.(*internal.ExportProfilesServiceRequest))
		return handler(ctx, req)
	}))
	received := make(chan ExportRequest, 1)
	RegisterGRPCServer(s, &capturingProfilesServer{received: received})
	wg := sync.WaitGroup{}
	wg.Go(func() {
		assert.NoError(t, s.Serve(lis))
	})
	t.Cleanup(func() {
		s.Stop()
		wg.Wait()
	})

	cc, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	profiles := pprofile.NewProfiles()
	resourceProfiles := profiles.ResourceProfiles().AppendEmpty()
	resourceProfiles.Resource().Attributes().PutStr("service.name", "checkout")
	resourceProfiles.Resource().Attributes().PutStr("empty.attr", "")
	scopeProfiles := resourceProfiles.ScopeProfiles().AppendEmpty()
	scopeProfiles.Scope().Attributes().PutStr("scope.attr", "scope-value")

	want := pprofile.NewProfiles()
	profiles.CopyTo(want)
	profiles.MarkReadOnly()
	want.MarkReadOnly()

	_, err = NewGRPCClient(cc).Export(context.Background(), NewExportRequestFromProfiles(profiles))
	require.NoError(t, err)

	// Read-only input is copied before the wire representation is prepared.
	assert.Equal(t, want, profiles)

	require.NotNil(t, wireRequest)
	assert.Equal(t, []string{"", "service.name", "checkout", "empty.attr", "scope.attr", "scope-value"}, wireRequest.Dictionary.StringTable)
	assertReferencedAttribute(t, wireRequest.ResourceProfiles[0].Resource.Attributes[0], 1, 2)
	assertReferencedAttribute(t, wireRequest.ResourceProfiles[0].Resource.Attributes[1], 3, 0)
	assertReferencedAttribute(t, wireRequest.ResourceProfiles[0].ScopeProfiles[0].Scope.Attributes[0], 4, 5)

	// The server resolves the references before exposing pdata to consumers.
	got := <-received
	resourceAttrs := got.Profiles().ResourceProfiles().At(0).Resource().Attributes()
	value, ok := resourceAttrs.Get("service.name")
	require.True(t, ok)
	assert.Equal(t, "checkout", value.Str())
	value, ok = resourceAttrs.Get("empty.attr")
	require.True(t, ok)
	assert.Empty(t, value.Str())
	scopeAttrs := got.Profiles().ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().Attributes()
	value, ok = scopeAttrs.Get("scope.attr")
	require.True(t, ok)
	assert.Equal(t, "scope-value", value.Str())
}

func assertReferencedAttribute(t *testing.T, attribute internal.KeyValue, keyIndex, valueIndex int32) {
	t.Helper()
	assert.Empty(t, attribute.Key)
	assert.Equal(t, keyIndex, attribute.KeyStrindex)
	reference, ok := attribute.Value.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, valueIndex, reference.StringValueStrindex)
}

type fakeProfilesServer struct {
	UnimplementedGRPCServer
	t   *testing.T
	err error
}

type capturingProfilesServer struct {
	UnimplementedGRPCServer
	received chan<- ExportRequest
	err      error
}

func (s capturingProfilesServer) Export(_ context.Context, request ExportRequest) (ExportResponse, error) {
	if s.received != nil {
		s.received <- request
	}
	return NewExportResponse(), s.err
}

func (f fakeProfilesServer) Export(_ context.Context, request ExportRequest) (ExportResponse, error) {
	assert.Equal(f.t, generateProfilesRequest(), request)
	return NewExportResponse(), f.err
}

func generateProfilesRequest() ExportRequest {
	td := pprofile.NewProfiles()
	td.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	return NewExportRequestFromProfiles(td)
}
