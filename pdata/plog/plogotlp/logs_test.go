// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plogotlp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	v1 "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/plog"
)

var _ json.Unmarshaler = Response{}
var _ json.Marshaler = Response{}

var _ json.Unmarshaler = Request{}
var _ json.Marshaler = Request{}

var logsRequestJSON = []byte(`
	{
		"resourceLogs": [
		{
			"resource": {},
			"scopeLogs": [
				{
					"scope": {},
					"logRecords": [
						{
							"body": {
								"stringValue": "test_log_record"
							},
							"traceId": "",
							"spanId": ""
						}
					]
				}
			]
		}
		]
	}`)

var logsTransitionData = [][]byte{
	[]byte(`
	{
		"resourceLogs": [
		{
			"resource": {},
			"instrumentationLibraryLogs": [
				{
					"instrumentationLibrary": {},
					"logRecords": [
						{
							"body": {
								"stringValue": "test_log_record"
							},
							"traceId": "",
							"spanId": ""
						}
					]
				}
			]
		}
		]
	}`),
	[]byte(`
	{
		"resourceLogs": [
		{
			"resource": {},
			"instrumentationLibraryLogs": [
				{
					"instrumentationLibrary": {},
					"logRecords": [
						{
							"body": {
								"stringValue": "test_log_record"
							},
							"traceId": "",
							"spanId": ""
						}
					]
				}
			],
			"scopeLogs": [
				{
					"scope": {},
					"logRecords": [
						{
							"body": {
								"stringValue": "test_log_record"
							},
							"traceId": "",
							"spanId": ""
						}
					]
				}
			]
		}
		]
	}`),
}

func TestRequestToPData(t *testing.T) {
	tr := NewRequest()
	assert.Equal(t, tr.Logs().LogRecordCount(), 0)
	tr.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	assert.Equal(t, tr.Logs().LogRecordCount(), 1)
}

func TestRequestJSON(t *testing.T) {
	lr := NewRequest()
	assert.NoError(t, lr.UnmarshalJSON(logsRequestJSON))
	assert.Equal(t, "test_log_record", lr.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsString())

	got, err := lr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(logsRequestJSON)), ""), string(got))
}

func TestRequestJSONTransition(t *testing.T) {
	for _, data := range logsTransitionData {
		lr := NewRequest()
		assert.NoError(t, lr.UnmarshalJSON(data))
		assert.Equal(t, "test_log_record", lr.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsString())

		got, err := lr.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, strings.Join(strings.Fields(string(logsRequestJSON)), ""), string(got))
	}
}

func TestGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterServer(s, &fakeLogsServer{t: t})
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

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewClient(cc)

	resp, err := logClient.Export(context.Background(), generateLogsRequest())
	assert.NoError(t, err)
	assert.Equal(t, NewResponse(), resp)
}

func TestGrpcTransition(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterServer(s, &fakeLogsServer{t: t})
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

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewClient(cc)

	req := generateLogsRequestWithInstrumentationLibrary()
	otlp.InstrumentationLibraryLogsToScope(req.orig.ResourceLogs)
	resp, err := logClient.Export(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, NewResponse(), resp)
}

type fakeRawServer struct {
	t *testing.T
}

func (s fakeRawServer) Export(_ context.Context, req Request) (Response, error) {
	assert.Equal(s.t, 1, req.Logs().LogRecordCount())
	return NewResponse(), nil
}

func TestGrpcExport(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterServer(s, &fakeRawServer{t: t})
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

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewClient(cc)

	resp, err := logClient.Export(context.Background(), generateLogsRequestWithInstrumentationLibrary())
	assert.NoError(t, err)
	assert.Equal(t, NewResponse(), resp)
}

func TestGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterServer(s, &fakeLogsServer{t: t, err: errors.New("my error")})
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

	cc, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cc.Close())
	})

	logClient := NewClient(cc)
	resp, err := logClient.Export(context.Background(), generateLogsRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, Response{}, resp)
}

type fakeLogsServer struct {
	t   *testing.T
	err error
}

func (f fakeLogsServer) Export(_ context.Context, request Request) (Response, error) {
	assert.Equal(f.t, generateLogsRequest(), request)
	return NewResponse(), f.err
}

func generateLogsRequest() Request {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStringVal("test_log_record")
	return NewRequestFromLogs(ld)
}

func generateLogsRequestWithInstrumentationLibrary() Request {
	lr := generateLogsRequest()
	lr.orig.ResourceLogs[0].InstrumentationLibraryLogs = []*v1.InstrumentationLibraryLogs{ //nolint:staticcheck // SA1019 ignore this!
		{
			LogRecords: lr.orig.ResourceLogs[0].ScopeLogs[0].LogRecords,
		},
	}
	lr.orig.ResourceLogs[0].ScopeLogs = []*v1.ScopeLogs{}
	return lr
}
