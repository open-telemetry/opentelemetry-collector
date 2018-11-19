// Copyright 2018, OpenCensus Authors
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

package exporterparser_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
	"github.com/census-instrumentation/opencensus-service/internal/testutils"
	"github.com/census-instrumentation/opencensus-service/receiver/zipkin"
)

// This function tests that Zipkin spans that are received then processed roundtrip
// back to almost the same JSON with differences:
// a) Go's net.IP.String intentional shortens 0s with "::" but also converts to hex values
//    so
//          "7::0.128.128.127"
//    becomes
//          "7::80:807f"
//
// The rest of the fields should match up exactly
func TestZipkinExportersFromYAML_roundtripJSON(t *testing.T) {
	responseReady := make(chan bool)
	buf := new(bytes.Buffer)
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		r.Body.Close()
		responseReady <- true
	}))
	defer cst.Close()

	config := `
exporters:
    zipkin:
        upload_period: 1ms
        endpoint: ` + cst.URL
	tes, doneFns, err := exporterparser.ZipkinExportersFromYAML([]byte(config))
	if err != nil {
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

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zi, err := zipkinreceiver.New(":0")
	if err != nil {
		t.Fatalf("Failed to create a new Zipkin receiver: %v", err)
	}

	zexp := exporter.MultiTraceExporters(tes...)
	if err := zi.StartTraceReception(context.Background(), zexp); err != nil {
		t.Fatalf("Failed to start trace reception: %v", err)
	}
	defer zi.StopTraceReception(context.Background())

	// Let the receiver receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)
	// Wait for the server to write the response.
	<-responseReady

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
}]`)

	// Finally we need to inspect the output
	gotBytes, _ := ioutil.ReadAll(buf)
	got := testutils.GenerateNormalizedJSON(string(gotBytes))
	if got != want {
		t.Errorf("RoundTrip result do not match:\nGot\n %s\n\nWant\n: %s\n", got, want)
	}
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
}]
`
