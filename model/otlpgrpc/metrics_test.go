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

package otlpgrpc

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

	v1 "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/model/internal/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

var _ json.Unmarshaler = MetricsResponse{}
var _ json.Marshaler = MetricsResponse{}

var _ json.Unmarshaler = MetricsRequest{}
var _ json.Marshaler = MetricsRequest{}

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

var metricsTransitionData = [][]byte{
	[]byte(`
		{
		"resourceMetrics": [
			{
				"resource": {},
				"instrumentationLibraryMetrics": [
					{
						"instrumentationLibrary": {},
						"metrics": [
							{
								"name": "test_metric"
							}
						]
					}
				]
			}
		]
		}`),
	[]byte(`
		{
		"resourceMetrics": [
			{
				"resource": {},
				"instrumentationLibraryMetrics": [
					{
						"instrumentationLibrary": {},
						"metrics": [
							{
								"name": "test_metric"
							}
						]
					}
				],
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
		}`),
}

func TestMetricsRequestJSON(t *testing.T) {
	mr := NewMetricsRequest()
	assert.NoError(t, mr.UnmarshalJSON(metricsRequestJSON))
	assert.Equal(t, "test_metric", mr.Metrics().ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())

	got, err := mr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(metricsRequestJSON)), ""), string(got))
}

func TestMetricsRequestJSONTransition(t *testing.T) {
	for _, data := range metricsTransitionData {
		mr := NewMetricsRequest()
		assert.NoError(t, mr.UnmarshalJSON(data))
		assert.Equal(t, "test_metric", mr.Metrics().ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())

		got, err := mr.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, strings.Join(strings.Fields(string(metricsRequestJSON)), ""), string(got))
	}
}

func TestMetricsGrpc(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterMetricsServer(s, &fakeMetricsServer{t: t})
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

	logClient := NewMetricsClient(cc)

	resp, err := logClient.Export(context.Background(), generateMetricsRequest())
	assert.NoError(t, err)
	assert.Equal(t, NewMetricsResponse(), resp)
}

func TestMetricsGrpcTransition(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterMetricsServer(s, &fakeMetricsServer{t: t})
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

	logClient := NewMetricsClient(cc)

	req := generateMetricsRequestWithInstrumentationLibrary()
	otlp.InstrumentationLibraryMetricsToScope(req.orig.ResourceMetrics)
	resp, err := logClient.Export(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, NewMetricsResponse(), resp)
}

func TestMetricsGrpcError(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterMetricsServer(s, &fakeMetricsServer{t: t, err: errors.New("my error")})
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

	logClient := NewMetricsClient(cc)
	resp, err := logClient.Export(context.Background(), generateMetricsRequest())
	require.Error(t, err)
	st, okSt := status.FromError(err)
	require.True(t, okSt)
	assert.Equal(t, "my error", st.Message())
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, MetricsResponse{}, resp)
}

type fakeMetricsServer struct {
	t   *testing.T
	err error
}

func (f fakeMetricsServer) Export(_ context.Context, request MetricsRequest) (MetricsResponse, error) {
	assert.Equal(f.t, generateMetricsRequest(), request)
	return NewMetricsResponse(), f.err
}

func generateMetricsRequest() MetricsRequest {
	md := pdata.NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetName("test_metric")

	mr := NewMetricsRequest()
	mr.SetMetrics(md)
	return mr
}

func generateMetricsRequestWithInstrumentationLibrary() MetricsRequest {
	mr := generateMetricsRequest()
	mr.orig.ResourceMetrics[0].InstrumentationLibraryMetrics = []*v1.InstrumentationLibraryMetrics{ //nolint:staticcheck // SA1019 ignore this!
		{
			Metrics: mr.orig.ResourceMetrics[0].ScopeMetrics[0].Metrics,
		},
	}
	return mr
}
