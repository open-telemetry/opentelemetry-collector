// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkinexporter

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinproto "github.com/openzipkin/zipkin-go/proto/v2"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
	"github.com/open-telemetry/opentelemetry-collector/translator/trace/zipkin"
)

func TestZipkinEndpointFromNode(t *testing.T) {
	type args struct {
		attributes   map[string]interface{}
		serviceName  string
		endpointType zipkinDirection
	}
	type want struct {
		endpoint      *zipkinmodel.Endpoint
		redundantKeys map[string]bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Nil attributes",
			args: args{attributes: nil, serviceName: "", endpointType: isLocalEndpoint},
			want: want{
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only svc name",
			args: args{
				attributes:   make(map[string]interface{}),
				serviceName:  "test",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test"},
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only ipv4",
			args: args{
				attributes:   map[string]interface{}{"ipv4": "1.2.3.4"},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv4: net.ParseIP("1.2.3.4")},
				redundantKeys: map[string]bool{"ipv4": true},
			},
		},
		{
			name: "Only ipv6 remote",
			args: args{
				attributes:   map[string]interface{}{zipkin.RemoteEndpointIPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
				serviceName:  "",
				endpointType: isRemoteEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv6: net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")},
				redundantKeys: map[string]bool{zipkin.RemoteEndpointIPv6: true},
			},
		},
		{
			name: "Only port",
			args: args{
				attributes:   map[string]interface{}{"port": "42"},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{Port: 42},
				redundantKeys: map[string]bool{"port": true},
			},
		},
		{
			name: "Service name, ipv4, and port",
			args: args{
				attributes:   map[string]interface{}{"ipv4": "4.3.2.1", "port": "2"},
				serviceName:  "test-svc",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test-svc", IPv4: net.ParseIP("4.3.2.1"), Port: 2},
				redundantKeys: map[string]bool{"ipv4": true, "port": true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redundantKeys := make(map[string]bool)
			endpoint := zipkinEndpointFromAttributes(
				tt.args.attributes,
				tt.args.serviceName,
				tt.args.endpointType,
				redundantKeys)
			assert.Equal(t, tt.want.endpoint, endpoint)
			assert.Equal(t, tt.want.redundantKeys, redundantKeys)
		})
	}
}

// This function tests that Zipkin spans that are received then processed roundtrip
// back to almost the same JSON with differences:
// a) Go's net.IP.String intentional shortens 0s with "::" but also converts to hex values
//    so
//          "7::0.128.128.127"
//    becomes
//          "7::80:807f"
//
// The rest of the fields should match up exactly
func TestZipkinExporter_roundtripJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		r.Body.Close()
	}))
	defer cst.Close()

	tes, err := newZipkinExporter(cst.URL, "", time.Millisecond, "json")
	assert.NoError(t, err)
	require.NotNil(t, tes)

	// The test requires the spans from zipkinSpansJSONJavaLibrary to be sent in a single batch, use
	// a mock to ensure that this happens as intended.
	mzr := newMockZipkinReporter(cst.URL)
	tes.reporter = mzr

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zexp := processor.NewTraceFanOutConnector([]consumer.TraceConsumer{tes})
	addr := testutils.GetAvailableLocalAddress(t)
	zi, err := zipkinreceiver.New(addr, zexp)
	assert.NoError(t, err)
	require.NotNil(t, zi)

	mh := component.NewMockHost()
	require.NoError(t, zi.Start(mh))
	defer zi.Shutdown()

	// Let the receiver receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)

	// Use the mock zipkin reporter to ensure all expected spans in a single batch. Since Flush waits for
	// server response there is no need for further synchronization.
	require.NoError(t, mzr.Flush())

	// We expect back the exact JSON that was received
	want := testutils.GenerateNormalizedJSON(t, `
[{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385","parentId": "86154a4ba6e91385","id": "4d1e00c0db9010db",
  "kind": "CLIENT","name": "get",
  "timestamp": 1472470996199000,"duration": 207000,
  "localEndpoint": {"serviceName": "frontend","ipv6": "7::80:807f"},
  "remoteEndpoint": {"serviceName": "backend","ipv4": "192.168.99.101","port": 9000},
  "annotations": [
    {"timestamp": 1472470996238000,"value": "foo"},
    {"timestamp": 1472470996403000,"value": "bar"}
  ],
  "tags": {"http.path": "/api","clnt/finagle.version": "6.45.0"}
},
{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385","parentId": "86154a4ba6e91386","id": "4d1e00c0db9010db",
  "kind": "SERVER","name": "put",
  "timestamp": 1472470996199000,"duration": 207000,
  "localEndpoint": {"serviceName": "frontend","ipv6": "7::80:807f"},
  "remoteEndpoint": {"serviceName": "frontend", "ipv4": "192.168.99.101","port": 9000},
  "annotations": [
    {"timestamp": 1472470996238000,"value": "foo"},
    {"timestamp": 1472470996403000,"value": "bar"}
  ],
  "tags": {"http.path": "/api","clnt/finagle.version": "6.45.0"}
},
{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
  "parentId": "86154a4ba6e91386",
  "id": "4d1e00c0db9010db",
  "kind": "SERVER",
  "name": "put",
  "timestamp": 1472470996199000,
  "duration": 207000
}]`)

	// Finally we need to inspect the output
	gotBytes, _ := ioutil.ReadAll(buf)
	got := testutils.GenerateNormalizedJSON(t, string(gotBytes))
	if got != want {
		t.Errorf("RoundTrip result do not match:\nGot\n %s\n\nWant\n: %s\n", got, want)
	}
}

type mockZipkinReporter struct {
	url        string
	client     *http.Client
	batch      []*zipkinmodel.SpanModel
	serializer zipkinreporter.SpanSerializer
}

var _ (zipkinreporter.Reporter) = (*mockZipkinReporter)(nil)

func (r *mockZipkinReporter) Send(span zipkinmodel.SpanModel) {
	r.batch = append(r.batch, &span)
}
func (r *mockZipkinReporter) Close() error {
	return nil
}

func newMockZipkinReporter(url string) *mockZipkinReporter {
	return &mockZipkinReporter{
		url:        url,
		client:     &http.Client{},
		serializer: zipkinreporter.JSONSerializer{},
	}
}

func (r *mockZipkinReporter) Flush() error {
	sendBatch := r.batch
	r.batch = nil

	if len(sendBatch) == 0 {
		return nil
	}

	body, err := r.serializer.Serialize(sendBatch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", r.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", r.serializer.ContentType())

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("http request failed with status code %d", resp.StatusCode)
	}

	return nil
}

const zipkinSpansJSONJavaLibrary = `
[{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
  "parentId": "86154a4ba6e91385",
  "id": "4d1e00c0db9010db",
  "kind": "CLIENT",
  "name": "get",
  "timestamp": 1472470996199000,
  "duration": 207000,
  "localEndpoint": {
    "serviceName": "frontend",
    "ipv6": "7::0.128.128.127"
  },
  "remoteEndpoint": {
    "serviceName": "backend",
    "ipv4": "192.168.99.101",
    "port": 9000
  },
  "annotations": [
    {
      "timestamp": 1472470996238000,
      "value": "foo"
    },
    {
      "timestamp": 1472470996403000,
      "value": "bar"
    }
  ],
  "tags": {
    "http.path": "/api",
    "clnt/finagle.version": "6.45.0"
  }
},
{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
  "parentId": "86154a4ba6e91386",
  "id": "4d1e00c0db9010db",
  "kind": "SERVER",
  "name": "put",
  "timestamp": 1472470996199000,
  "duration": 207000,
  "localEndpoint": {
    "serviceName": "frontend",
    "ipv6": "7::0.128.128.127"
  },
  "remoteEndpoint": {
    "serviceName": "frontend",
    "ipv4": "192.168.99.101",
    "port": 9000
  },
  "annotations": [
    {
      "timestamp": 1472470996238000,
      "value": "foo"
    },
    {
      "timestamp": 1472470996403000,
      "value": "bar"
    }
  ],
  "tags": {
    "http.path": "/api",
    "clnt/finagle.version": "6.45.0"
  }
},
{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
  "parentId": "86154a4ba6e91386",
  "id": "4d1e00c0db9010db",
  "kind": "SERVER",
  "name": "put",
  "timestamp": 1472470996199000,
  "duration": 207000
}]
`

func TestZipkinExporter_invalidFormat(t *testing.T) {
	_, err := newZipkinExporter("", "", 0, "foobar")
	require.Error(t, err)
}

// The rest of the fields should match up exactly
func TestZipkinExporter_roundtripProto(t *testing.T) {
	buf := new(bytes.Buffer)
	var contentType string
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		contentType = r.Header.Get("Content-Type")
		r.Body.Close()
	}))
	defer cst.Close()

	tes, err := newZipkinExporter(cst.URL, "", time.Millisecond, "proto")
	require.NoError(t, err)

	// The test requires the spans from zipkinSpansJSONJavaLibrary to be sent in a single batch, use
	// a mock to ensure that this happens as intended.
	mzr := newMockZipkinReporter(cst.URL)
	tes.reporter = mzr

	mzr.serializer = zipkinproto.SpanSerializer{}

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zexp := processor.NewTraceFanOutConnector([]consumer.TraceConsumer{tes})
	port := testutils.GetAvailablePort(t)
	zi, err := zipkinreceiver.New(fmt.Sprintf(":%d", port), zexp)
	require.NoError(t, err)

	mh := component.NewMockHost()
	err = zi.Start(mh)
	require.NoError(t, err)
	defer zi.Shutdown()

	// Let the receiver receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)

	// Use the mock zipkin reporter to ensure all expected spans in a single batch. Since Flush waits for
	// server response there is no need for further synchronization.
	err = mzr.Flush()
	require.NoError(t, err)

	require.Equal(t, zipkinproto.SpanSerializer{}.ContentType(), contentType)
	// Finally we need to inspect the output
	gotBytes, err := ioutil.ReadAll(buf)
	require.NoError(t, err)

	_, err = zipkinproto.ParseSpans(gotBytes, false)
	require.NoError(t, err)

}
