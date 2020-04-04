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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinproto "github.com/openzipkin/zipkin-go/proto/v2"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
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
func TestZipkinExporter_roundtripJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	sizes := []int64{}
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, _ := io.Copy(buf, r.Body)
		sizes = append(sizes, s)
		r.Body.Close()
	}))
	defer cst.Close()

	config := &Config{
		URL:    cst.URL,
		Format: "json",
	}
	tes, err := NewTraceExporter(zap.NewNop(), config)
	assert.NoError(t, err)
	require.NotNil(t, tes)

	// The test requires the spans from zipkinSpansJSONJavaLibrary to be sent in a single batch, use
	// a mock to ensure that this happens as intended.
	mzr := newMockZipkinReporter(cst.URL)

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zexp := processor.NewTraceFanOutConnectorOld([]consumer.TraceConsumerOld{tes})
	addr := testutils.GetAvailableLocalAddress(t)
	zi, err := zipkinreceiver.New("zipkin_receiver", addr, zexp)
	assert.NoError(t, err)
	require.NotNil(t, zi)

	require.NoError(t, zi.Start(context.Background(), componenttest.NewNopHost()))
	defer zi.Shutdown(context.Background())

	// Let the receiver receive "uploaded Zipkin spans from a Java client application"
	req, _ := http.NewRequest("POST", "https://tld.org/", strings.NewReader(zipkinSpansJSONJavaLibrary))
	responseWriter := httptest.NewRecorder()
	zi.ServeHTTP(responseWriter, req)

	// Use the mock zipkin reporter to ensure all expected spans in a single batch. Since Flush waits for
	// server response there is no need for further synchronization.
	require.NoError(t, mzr.Flush())

	// We expect back the exact JSON that was received
	wants := []string{`
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
		}]
		`, `
		[{
		  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
		  "parentId": "86154a4ba6e91386",
		  "id": "4d1e00c0db9010db",
		  "kind": "SERVER",
		  "name": "put",
		  "timestamp": 1472470996199000,
		  "duration": 207000
		}]
		`}
	for i, s := range wants {
		want := testutils.GenerateNormalizedJSON(t, s)
		gotBytes := buf.Next(int(sizes[i]))
		got := testutils.GenerateNormalizedJSON(t, string(gotBytes))
		assert.Equal(t, want, got)
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
	config := &Config{
		URL:    "1.2.3.4",
		Format: "foobar",
	}
	f := &Factory{}
	_, err := f.CreateTraceExporter(zap.NewNop(), config)
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

	config := &Config{
		URL:    cst.URL,
		Format: "proto",
	}
	tes, err := NewTraceExporter(zap.NewNop(), config)
	require.NoError(t, err)

	// The test requires the spans from zipkinSpansJSONJavaLibrary to be sent in a single batch, use
	// a mock to ensure that this happens as intended.
	mzr := newMockZipkinReporter(cst.URL)

	mzr.serializer = zipkinproto.SpanSerializer{}

	// Run the Zipkin receiver to "receive spans upload from a client application"
	zexp := processor.NewTraceFanOutConnectorOld([]consumer.TraceConsumerOld{tes})
	port := testutils.GetAvailablePort(t)
	zi, err := zipkinreceiver.New(
		"zipkin_receiver", fmt.Sprintf(":%d", port), zexp)
	require.NoError(t, err)

	err = zi.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer zi.Shutdown(context.Background())

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
