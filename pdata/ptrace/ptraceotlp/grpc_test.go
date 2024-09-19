// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp

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

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeTracesServer{t: t})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, s.Serve(lis))
	}()
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

	resp, err := logClient.Export(context.Background(), generateTracesRequest())
	require.NoError(t, err)
	assert.Equal(t, NewExportResponse(), resp)
}

func TestGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeTracesServer{t: t, err: errors.New("my error")})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, s.Serve(lis))
	}()
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
	resp, err := logClient.Export(context.Background(), generateTracesRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, ExportResponse{}, resp)
}

type fakeTracesServer struct {
	UnimplementedGRPCServer
	t   *testing.T
	err error
}

func (f fakeTracesServer) Export(_ context.Context, request ExportRequest) (ExportResponse, error) {
	assert.Equal(f.t, generateTracesRequest(), request)
	return NewExportResponse(), f.err
}

func generateTracesRequest() ExportRequest {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("test_span")
	return NewExportRequestFromTraces(td)
}
