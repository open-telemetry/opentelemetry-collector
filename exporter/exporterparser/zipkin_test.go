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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
	"github.com/census-instrumentation/opencensus-service/interceptor/zipkin"
	"github.com/census-instrumentation/opencensus-service/internal/testutils"
)

// This function tests that Zipkin spans that are intercepted then processed roundtrip
// back to almost the same JSON with differences:
// a) Go's net.IP.String intentional shortens 0s with "::" but also converts to hex values
//    so
//          "7::0.128.128.127"
//    becomes
//          "7::80:807f"
//
// The rest of the fields should match up exactly
func TestZipkinExportersFromYAML_roundtripJSON(t *testing.T) {
	cb := newConcurrentBuffer()
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(cb, r.Body)
		r.Body.Close()
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

	zexp := exporter.MultiTraceExporters(tes...)

	// Run the Zipkin interceptor to "receive spans upload from a client application"
	zi, err := zipkininterceptor.New(zexp)
	if err != nil {
		t.Fatalf("Failed to create a new Zipkin interceptor: %v", err)
	}

	// Let the interceptor receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)
	// Just for 100X longer than the Zipkin reporting period
	// because -race slows things down a lot.
	<-time.After(100 * time.Millisecond)

	// We expect back the exact JSON that was intercepted
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
	gotBytes, _ := ioutil.ReadAll(cb)
	got := testutils.GenerateNormalizedJSON(string(gotBytes))
	if got != want {
		t.Errorf("RoundTrip result do not match:\nGot\n %s\n\nWant\n: %s\n", got, want)
	}
}

type concurrentBuffer struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func newConcurrentBuffer() *concurrentBuffer {
	return &concurrentBuffer{
		buf: new(bytes.Buffer),
	}
}

func (cb *concurrentBuffer) Write(b []byte) (int, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.buf.Write(b)
}

func (cb *concurrentBuffer) Read(b []byte) (int, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.buf.Read(b)
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
