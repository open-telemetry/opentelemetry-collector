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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"

	"github.com/open-telemetry/opentelemetry-service/internal/config/viperutils"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/processor/multiconsumer"
	"github.com/open-telemetry/opentelemetry-service/receiver/zipkinreceiver"
)

func TestZipkinEndpointFromNode(t *testing.T) {
	type args struct {
		node         *commonpb.Node
		serviceName  string
		endpointType zipkinDirection
	}
	tests := []struct {
		name    string
		args    args
		want    *zipkinmodel.Endpoint
		wantErr bool
	}{
		{
			name:    "Nil Node",
			args:    args{node: nil, serviceName: "", endpointType: isLocalEndpoint},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "Only svc name",
			args:    args{node: &commonpb.Node{}, serviceName: "test", endpointType: isLocalEndpoint},
			want:    &zipkinmodel.Endpoint{ServiceName: "test"},
			wantErr: false,
		},
		{
			name: "Only ipv4",
			args: args{
				node: &commonpb.Node{
					Attributes: map[string]string{"ipv4": "1.2.3.4"},
				},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want:    &zipkinmodel.Endpoint{IPv4: net.ParseIP("1.2.3.4")},
			wantErr: false,
		},
		{
			name: "Only ipv6 remote",
			args: args{
				node: &commonpb.Node{
					Attributes: map[string]string{"zipkin.remoteEndpoint.ipv6": "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
				},
				serviceName:  "",
				endpointType: isRemoteEndpoint,
			},
			want:    &zipkinmodel.Endpoint{IPv6: net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")},
			wantErr: false,
		},
		{
			name: "Only port",
			args: args{
				node: &commonpb.Node{
					Attributes: map[string]string{"port": "42"},
				},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want:    &zipkinmodel.Endpoint{Port: 42},
			wantErr: false,
		},
		{
			name: "Service name, ipv4, and port",
			args: args{
				node: &commonpb.Node{
					Attributes: map[string]string{"ipv4": "4.3.2.1", "port": "2"},
				},
				serviceName:  "test-svc",
				endpointType: isLocalEndpoint,
			},
			want:    &zipkinmodel.Endpoint{ServiceName: "test-svc", IPv4: net.ParseIP("4.3.2.1"), Port: 2},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := zipkinEndpointFromNode(tt.args.node, tt.args.serviceName, tt.args.endpointType)
			if (err != nil) != tt.wantErr {
				t.Errorf("zipkinEndpointFromNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zipkinEndpointFromNode() = %v, want %v", got, tt.want)
			}
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
func TestZipkinExportersFromViper_roundtripJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		r.Body.Close()
	}))
	defer cst.Close()

	config := `
zipkin:
  upload_period: 1ms
  endpoint: ` + cst.URL
	v, _ := viperutils.ViperFromYAMLBytes([]byte(config))
	tes, _, doneFns, err := ZipkinExportersFromViper(v)
	if len(tes) == 0 || err != nil {
		t.Fatalf("Failed to parse out exporters: %v", err)
	}
	defer func() {
		for _, fn := range doneFns {
			fn()
		}
	}()

	if g, w := len(tes), 1; g != w {
		t.Errorf("Number of trace exporters: Got %d Want %d", g, w)
	}

	// The test requires the spans from zipkinSpansJSONJavaLibrary to be sent in a single batch, use
	// a mock to ensure that this happens as intended.
	mzr := newMockZipkinReporter(cst.URL)
	tes[0].(*zipkinExporter).reporter = mzr

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zexp := multiconsumer.NewTraceProcessor(tes)
	zi, err := zipkinreceiver.New(":0", zexp)
	if err != nil {
		t.Fatalf("Failed to create a new Zipkin receiver: %v", err)
	}

	if err := zi.StartTraceReception(context.Background(), nil); err != nil {
		t.Fatalf("Failed to start trace reception: %v", err)
	}
	defer zi.StopTraceReception(context.Background())

	// Let the receiver receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)

	// Use the mock zipkin reporter to ensure all expected spans in a single batch. Since Flush waits for
	// server response there is no need for further synchronization.
	if err := mzr.Flush(); err != nil {
		t.Fatalf("Failed to flush zipkin reporter: %v", err)
	}

	// We expect back the exact JSON that was received
	want := testutils.GenerateNormalizedJSON(`
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
	got := testutils.GenerateNormalizedJSON(string(gotBytes))
	if got != want {
		t.Errorf("RoundTrip result do not match:\nGot\n %s\n\nWant\n: %s\n", got, want)
	}
}

type mockZipkinReporter struct {
	url    string
	client *http.Client
	batch  []*zipkinmodel.SpanModel
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
		url:    url,
		client: &http.Client{},
	}
}

func (r *mockZipkinReporter) Flush() error {
	sendBatch := r.batch
	r.batch = nil

	if len(sendBatch) == 0 {
		return nil
	}

	body, err := json.Marshal(sendBatch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", r.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

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
