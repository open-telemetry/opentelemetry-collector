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

package pmetricotlp

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

	"go.opentelemetry.io/collector/pdata/pmetric"
)

var _ json.Unmarshaler = Response{}
var _ json.Marshaler = Response{}

var _ json.Unmarshaler = Request{}
var _ json.Marshaler = Request{}

var metricsRequestJSON = []byte(`
	{
		"resourceMetrics": [
			{
				"resource": {},
				"scopeMetrics": [
					{
						"scope": {},
						"metrics": [
							{
								"name": "test_metric"
							}
						]
					}
				]
			}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewRequest()
	assert.Equal(t, tr.Metrics().MetricCount(), 0)
	tr.Metrics().ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	assert.Equal(t, tr.Metrics().MetricCount(), 1)
}

func TestRequestJSON(t *testing.T) {
	mr := NewRequest()
	assert.NoError(t, mr.UnmarshalJSON(metricsRequestJSON))
	assert.Equal(t, "test_metric", mr.Metrics().ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())

	got, err := mr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(metricsRequestJSON)), ""), string(got))
}

func TestGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeMetricsServer{t: t})
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

	resp, err := logClient.Export(context.Background(), generateMetricsRequest())
	assert.NoError(t, err)
	assert.Equal(t, NewResponse(), resp)
}

func TestGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterGRPCServer(s, &fakeMetricsServer{t: t, err: errors.New("my error")})
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
	resp, err := logClient.Export(context.Background(), generateMetricsRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, Response{}, resp)
}

type fakeMetricsServer struct {
	t   *testing.T
	err error
}

func (f fakeMetricsServer) Export(_ context.Context, request Request) (Response, error) {
	assert.Equal(f.t, generateMetricsRequest(), request)
	return NewResponse(), f.err
}

func generateMetricsRequest() Request {
	md := pmetric.NewMetrics()
	m := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetName("test_metric")
	m.SetEmptyGauge().DataPoints().AppendEmpty()
	return NewRequestFromMetrics(md)
}
