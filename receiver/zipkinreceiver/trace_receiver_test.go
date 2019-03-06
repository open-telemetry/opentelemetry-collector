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

package zipkinreceiver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter/exportertest"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/internal/testutils"
	spandatatranslator "github.com/census-instrumentation/opencensus-service/translator/trace/spandata"
)

func TestTraceIDConversion(t *testing.T) {
	longID, _ := zipkinmodel.TraceIDFromHex("01020304050607080102030405060708")
	shortID, _ := zipkinmodel.TraceIDFromHex("0102030405060708")
	zeroID, _ := zipkinmodel.TraceIDFromHex("0000000000000000")
	tests := []struct {
		name    string
		id      zipkinmodel.TraceID
		want    []byte
		wantErr error
	}{
		{
			name:    "128bit traceID",
			id:      longID,
			want:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
			wantErr: nil,
		},
		{
			name:    "64bit traceID",
			id:      shortID,
			want:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8},
			wantErr: nil,
		},
		{
			name:    "zero traceID",
			id:      zeroID,
			want:    nil,
			wantErr: errZeroTraceID,
		},
	}

	for _, tc := range tests {
		got, gotErr := zTraceIDToOCProtoTraceID(tc.id)
		if tc.wantErr != gotErr {
			t.Errorf("gotErr=%v wantErr=%v", gotErr, tc.wantErr)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got=%v want=%v", got, tc.want)
		}
	}
}

func TestShortIDSpanConversion(t *testing.T) {
	shortID, _ := zipkinmodel.TraceIDFromHex("0102030405060708")
	if shortID.High != 0 {
		t.Errorf("wanted 64bit traceID, so TraceID.High must be zero")
	}

	zc := zipkinmodel.SpanContext{
		TraceID: shortID,
		ID:      zipkinmodel.ID(shortID.Low),
	}
	zs := zipkinmodel.SpanModel{
		SpanContext: zc,
	}

	ocSpan, _, err := zipkinSpanToTraceSpan(&zs)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(ocSpan.TraceId) != 16 {
		t.Fatalf("incorrect OC proto trace id length")
	}

	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}
	if !reflect.DeepEqual(ocSpan.TraceId, want) {
		t.Errorf("got=%v want=%v", ocSpan.TraceId, want)
	}
}

func TestConvertSpansToTraceSpans_json(t *testing.T) {
	// Using Adrian Cole's sample at https://gist.github.com/adriancole/e8823c19dfed64e2eb71
	blob, err := ioutil.ReadFile("./testdata/sample1.json")
	if err != nil {
		t.Fatalf("Failed to read sample JSON file: %v", err)
	}
	zi := new(ZipkinReceiver)
	reqs, err := zi.v2ToTraceSpans(blob, nil)
	if err != nil {
		t.Fatalf("Failed to parse convert Zipkin spans in JSON to Trace spans: %v", err)
	}

	if g, w := len(reqs), 1; g != w {
		t.Fatalf("Expecting only one request since all spans share same node/localEndpoint: %v", g)
	}

	req := reqs[0]
	wantNode := &commonpb.Node{
		ServiceInfo: &commonpb.ServiceInfo{
			Name: "frontend",
		},
		Attributes: map[string]string{
			"ipv6":                              "7::80:807f",
			"serviceName":                       "frontend",
			"zipkin.remoteEndpoint.serviceName": "backend",
			"zipkin.remoteEndpoint.ipv4":        "192.168.99.101",
			"zipkin.remoteEndpoint.port":        "9000",
		},
	}
	if g, w := req.Node, wantNode; !reflect.DeepEqual(g, w) {
		t.Errorf("GotNode:\n\t%v\nWantNode:\n\t%v", g, w)
	}

	nonNilSpans := 0
	for _, span := range req.Spans {
		if span != nil {
			nonNilSpans++
		}
	}
	// Expecting 9 non-nil spans
	if g, w := nonNilSpans, 9; g != w {
		t.Fatalf("Non-nil spans: Got %d Want %d", g, w)
	}
}

func TestConversionRoundtrip(t *testing.T) {
	// The goal is to convert from:
	// 1. Original Zipkin JSON as that's the format that Zipkin receivers will receive
	// 2. Into TraceProtoSpans
	// 3. Into SpanData
	// 4. Back into Zipkin JSON (in this case the Zipkin exporter has been configured)
	receiverInputJSON := []byte(`
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
}]`)

	zi := &ZipkinReceiver{nextConsumer: exportertest.NewNopTraceExporter()}
	ereqs, err := zi.v2ToTraceSpans(receiverInputJSON, nil)
	if err != nil {
		t.Fatalf("Failed to parse and convert receiver JSON: %v", err)
	}

	wantProtoRequests := []data.TraceData{
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "frontend"},
				Attributes: map[string]string{
					"ipv6":                              "7::80:807f",
					"serviceName":                       "frontend",
					"zipkin.remoteEndpoint.serviceName": "backend",
					"zipkin.remoteEndpoint.ipv4":        "192.168.99.101",
					"zipkin.remoteEndpoint.port":        "9000",
				},
			},

			Spans: []*tracepb.Span{
				{
					TraceId:      []byte{0x4d, 0x1e, 0x00, 0xc0, 0xdb, 0x90, 0x10, 0xdb, 0x86, 0x15, 0x4a, 0x4b, 0xa6, 0xe9, 0x13, 0x85},
					ParentSpanId: []byte{0x86, 0x15, 0x4a, 0x4b, 0xa6, 0xe9, 0x13, 0x85},
					SpanId:       []byte{0x4d, 0x1e, 0x00, 0xc0, 0xdb, 0x90, 0x10, 0xdb},
					Kind:         tracepb.Span_CLIENT,
					Name:         &tracepb.TruncatableString{Value: "get"},
					StartTime:    internal.TimeToTimestamp(time.Unix(int64(1472470996199000)/1e6, 1e3*(int64(1472470996199000)%1e6))),
					EndTime:      internal.TimeToTimestamp(time.Unix(int64(1472470996199000+207000)/1e6, 1e3*(int64(1472470996199000+207000)%1e6))),
					TimeEvents: &tracepb.Span_TimeEvents{
						TimeEvent: []*tracepb.Span_TimeEvent{
							{
								Time: internal.TimeToTimestamp(time.Unix(int64(1472470996238000)/1e6, 1e3*(int64(1472470996238000)%1e6))),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{Value: "foo"},
									},
								},
							},
							{
								Time: internal.TimeToTimestamp(time.Unix(int64(1472470996403000)/1e6, 1e3*(int64(1472470996403000)%1e6))),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{Value: "bar"},
									},
								},
							},
						},
					},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"http.path": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "/api"},
								},
							},
							"clnt/finagle.version": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "6.45.0"},
								},
							},
						},
					},
				},
			},
		},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "frontend"},
				Attributes: map[string]string{
					"ipv6":                              "7::80:807f",
					"serviceName":                       "frontend",
					"zipkin.remoteEndpoint.serviceName": "frontend",
					"zipkin.remoteEndpoint.ipv4":        "192.168.99.101",
					"zipkin.remoteEndpoint.port":        "9000",
				},
			},

			Spans: []*tracepb.Span{
				{
					TraceId:      []byte{0x4d, 0x1e, 0x00, 0xc0, 0xdb, 0x90, 0x10, 0xdb, 0x86, 0x15, 0x4a, 0x4b, 0xa6, 0xe9, 0x13, 0x85},
					SpanId:       []byte{0x4d, 0x1e, 0x00, 0xc0, 0xdb, 0x90, 0x10, 0xdb},
					ParentSpanId: []byte{0x86, 0x15, 0x4a, 0x4b, 0xa6, 0xe9, 0x13, 0x86},
					Kind:         tracepb.Span_SERVER,
					Name:         &tracepb.TruncatableString{Value: "put"},
					StartTime:    internal.TimeToTimestamp(time.Unix(int64(1472470996199000)/1e6, 1e3*(int64(1472470996199000)%1e6))),
					EndTime:      internal.TimeToTimestamp(time.Unix(int64(1472470996199000+207000)/1e6, 1e3*(int64(1472470996199000+207000)%1e6))),
					TimeEvents: &tracepb.Span_TimeEvents{
						TimeEvent: []*tracepb.Span_TimeEvent{
							{
								Time: internal.TimeToTimestamp(time.Unix(int64(1472470996238000)/1e6, 1e3*(int64(1472470996238000)%1e6))),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{Value: "foo"},
									},
								},
							},
							{
								Time: internal.TimeToTimestamp(time.Unix(int64(1472470996403000)/1e6, 1e3*(int64(1472470996403000)%1e6))),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{Value: "bar"},
									},
								},
							},
						},
					},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"http.path": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "/api"},
								},
							},
							"clnt/finagle.version": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "6.45.0"},
								},
							},
						},
					},
				},
			},
		},
	}

	g, w := ereqs, wantProtoRequests
	if len(g) != len(w) {
		t.Errorf("Unmatched lengths:\nGot: %d\nWant: %d", len(g), len(w))
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("Failed to transform the expected ProtoSpans\nGot:\n\t%v\nWant:\n\t%v\n", g, w)
	}

	// Now the last phase is to transmit them over the wire and then compare the JSONs

	buf := new(bytes.Buffer)
	// This will act as the final Zipkin server.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		_ = r.Body.Close()
	}))
	defer backend.Close()

	var reporterShutdownFns []func() error
	for _, treq := range ereqs {
		ipv6 := treq.Node.Attributes["ipv6"]
		// Convert this ipv6 address to hostport as recommended in:
		//      https://github.com/openzipkin/zipkin-go/issues/84
		hostPort := fmt.Sprintf("[%s]:0", ipv6)
		le, err := openzipkin.NewEndpoint(treq.Node.GetServiceInfo().GetName(), hostPort)
		if err != nil {
			t.Errorf("NewEndpoint err: %v", err)
			continue
		}
		re := zhttp.NewReporter(backend.URL)
		ze := zipkin.NewExporter(re, le)
		reporterShutdownFns = append(reporterShutdownFns, re.Close)

		for _, span := range treq.Spans {
			sd, _ := spandatatranslator.ProtoSpanToOCSpanData(span)
			ze.ExportSpan(sd)
		}
	}
	// Now shut down the respective reporters and the server
	for _, reporterFn := range reporterShutdownFns {
		_ = reporterFn()
	}
	backend.Close()

	// Give them time to shutdown
	<-time.After(200 * time.Millisecond)

	// We expect this final JSON: that is
	wantFinalJSON := `
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
    "ipv6": "7::80:807f"
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
[{
  "traceId": "4d1e00c0db9010db86154a4ba6e91385",
  "parentId": "86154a4ba6e91386",
  "id": "4d1e00c0db9010db",
  "kind": "SERVER",
  "name": "put",
  "timestamp": 1472470996199000,
  "duration": 207000,
  "localEndpoint": {
    "serviceName": "frontend",
    "ipv6": "7::80:807f"
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
}]`

	gj, wj := testutils.GenerateNormalizedJSON(buf.String()), testutils.GenerateNormalizedJSON(wantFinalJSON)
	if gj != wj {
		t.Errorf("The roundtrip JSON doesn't match the JSON that we want\nGot:\n%s\nWant:\n%s", gj, wj)
	}
}
