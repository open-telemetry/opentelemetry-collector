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

package zipkinreceiver

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	zipkin2 "github.com/jaegertracing/jaeger/model/converter/thrift/zipkin"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/conventions"
)

const zipkinReceiverName = "zipkin_receiver_test"

func TestNew(t *testing.T) {
	type args struct {
		address      string
		nextConsumer consumer.TracesConsumer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name:    "nil nextConsumer",
			args:    args{},
			wantErr: componenterror.ErrNilNextConsumer,
		},
		{
			name: "happy path",
			args: args{
				nextConsumer: consumertest.NewTracesNop(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					NameVal: zipkinReceiverName,
				},
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: tt.args.address,
				},
			}
			got, err := New(cfg, tt.args.nextConsumer)
			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.NotNil(t, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}

func TestZipkinReceiverPortAlreadyInUse(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "failed to open a port: %v", err)
	defer l.Close()
	_, portStr, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err, "failed to split listener address: %v", err)
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: zipkinReceiverName,
		},
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: "localhost:" + portStr,
		},
	}
	traceReceiver, err := New(cfg, consumertest.NewTracesNop())
	require.NoError(t, err, "Failed to create receiver: %v", err)
	err = traceReceiver.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err)
}

func TestConvertSpansToTraceSpans_json(t *testing.T) {
	// Using Adrian Cole's sample at https://gist.github.com/adriancole/e8823c19dfed64e2eb71
	blob, err := ioutil.ReadFile("./testdata/sample1.json")
	require.NoError(t, err, "Failed to read sample JSON file: %v", err)
	zi := new(ZipkinReceiver)
	zi.config = createDefaultConfig().(*Config)
	reqs, err := zi.v2ToTraceSpans(blob, nil)
	require.NoError(t, err, "Failed to parse convert Zipkin spans in JSON to Trace spans: %v", err)

	require.Equal(t, 1, reqs.ResourceSpans().Len(), "Expecting only one request since all spans share same node/localEndpoint: %v", reqs.ResourceSpans().Len())

	req := reqs.ResourceSpans().At(0)
	sn, _ := req.Resource().Attributes().Get(conventions.AttributeServiceName)
	assert.Equal(t, "frontend", sn.StringVal())

	// Expecting 9 non-nil spans
	require.Equal(t, 9, reqs.SpanCount(), "Incorrect non-nil spans count")
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
    "ipv6": "7::80:807f"
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
    "clnt/finagle.version": "6.45.0",
	"status.code": "STATUS_CODE_UNSET"
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
    "ipv6": "7::80:807f"
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
    "clnt/finagle.version": "6.45.0",
	"status.code": "STATUS_CODE_UNSET"
  }
}]`)

	zi := &ZipkinReceiver{nextConsumer: consumertest.NewTracesNop()}
	zi.config = &Config{}
	ereqs, err := zi.v2ToTraceSpans(receiverInputJSON, nil)
	require.NoError(t, err)

	require.Equal(t, 2, ereqs.SpanCount())

	// Now the last phase is to transmit them over the wire and then compare the JSONs

	buf := new(bytes.Buffer)
	// This will act as the final Zipkin server.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(buf, r.Body)
		_ = r.Body.Close()
	}))
	defer backend.Close()

	factory := zipkinexporter.NewFactory()
	config := factory.CreateDefaultConfig().(*zipkinexporter.Config)
	config.Endpoint = backend.URL
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	ze, err := factory.CreateTracesExporter(context.Background(), params, config)
	require.NoError(t, err)
	require.NotNil(t, ze)
	require.NoError(t, ze.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, ze.ConsumeTraces(context.Background(), ereqs))

	// Shutdown the exporter so it can flush any remaining data.
	assert.NoError(t, ze.Shutdown(context.Background()))
	backend.Close()

	// The received JSON messages are inside arrays, so reading then directly will
	// fail with error. Use a small hack to transform the multiple arrays into a
	// single one.
	accumulatedJSONMsgs := strings.ReplaceAll(buf.String(), "][", ",")
	gj := testutil.GenerateNormalizedJSON(t, accumulatedJSONMsgs)
	wj := testutil.GenerateNormalizedJSON(t, string(receiverInputJSON))
	// translation to OTLP sorts spans so do a span-by-span comparison
	gj = gj[1 : len(gj)-1]
	wj = wj[1 : len(wj)-1]
	gjSpans := strings.Split(gj, "{\"annotations\":")
	wjSpans := strings.Split(wj, "{\"annotations\":")
	assert.Equal(t, len(wjSpans), len(gjSpans))
	for _, wjspan := range wjSpans {
		if len(wjspan) > 3 && wjspan[len(wjspan)-1:] == "," {
			wjspan = wjspan[0 : len(wjspan)-1]
		}
		matchFound := false
		for _, gjspan := range gjSpans {
			if len(gjspan) > 3 && gjspan[len(gjspan)-1:] == "," {
				gjspan = gjspan[0 : len(gjspan)-1]
			}
			if wjspan == gjspan {
				matchFound = true
			}
		}
		assert.True(t, matchFound, fmt.Sprintf("no match found for {\"annotations\":%s %v", wjspan, gjSpans))
	}
}

func TestStartTraceReception(t *testing.T) {
	tests := []struct {
		name    string
		host    component.Host
		wantErr bool
	}{
		{
			name:    "nil_host",
			wantErr: true,
		},
		{
			name: "valid_host",
			host: componenttest.NewNopHost(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			cfg := &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					NameVal: zipkinReceiverName,
				},
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:0",
				},
			}
			zr, err := New(cfg, sink)
			require.Nil(t, err)
			require.NotNil(t, zr)

			err = zr.Start(context.Background(), tt.host)
			assert.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				require.Nil(t, zr.Shutdown(context.Background()))
			}
		})
	}
}

func TestReceiverContentTypes(t *testing.T) {
	tests := []struct {
		endpoint string
		content  string
		encoding string
		bodyFn   func() ([]byte, error)
	}{
		{
			endpoint: "/api/v1/spans",
			content:  "application/json",
			encoding: "gzip",
			bodyFn: func() ([]byte, error) {
				return ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v1_single_batch.json")
			},
		},

		{
			endpoint: "/api/v1/spans",
			content:  "application/x-thrift",
			encoding: "gzip",
			bodyFn: func() ([]byte, error) {
				return thriftExample(), nil
			},
		},

		{
			endpoint: "/api/v2/spans",
			content:  "application/json",
			encoding: "gzip",
			bodyFn: func() ([]byte, error) {
				return ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v2_single.json")
			},
		},

		{
			endpoint: "/api/v2/spans",
			content:  "application/json",
			encoding: "zlib",
			bodyFn: func() ([]byte, error) {
				return ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v2_single.json")
			},
		},

		{
			endpoint: "/api/v2/spans",
			content:  "application/json",
			encoding: "",
			bodyFn: func() ([]byte, error) {
				return ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v2_single.json")
			},
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("%v %v %v", test.endpoint, test.content, test.encoding)
		t.Run(name, func(t *testing.T) {
			body, err := test.bodyFn()
			require.NoError(t, err, "Failed to generate test body: %v", err)

			var requestBody *bytes.Buffer
			switch test.encoding {
			case "":
				requestBody = bytes.NewBuffer(body)
			case "zlib":
				requestBody, err = compressZlib(body)
			case "gzip":
				requestBody, err = compressGzip(body)
			}
			require.NoError(t, err)

			r := httptest.NewRequest("POST", test.endpoint, requestBody)
			r.Header.Add("content-type", test.content)
			r.Header.Add("content-encoding", test.encoding)

			next := &zipkinMockTraceConsumer{
				ch: make(chan pdata.Traces, 10),
			}
			cfg := &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					NameVal: zipkinReceiverName,
				},
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "",
				},
			}
			zr, err := New(cfg, next)
			require.NoError(t, err)

			req := httptest.NewRecorder()
			zr.ServeHTTP(req, r)

			select {
			case td := <-next.ch:
				require.NotNil(t, td)
				require.Equal(t, 202, req.Code)
				break
			case <-time.After(time.Second * 2):
				t.Error("next consumer did not receive the batch")
				break
			}
		})
	}
}

func TestReceiverInvalidContentType(t *testing.T) {
	body := `{ invalid json `

	r := httptest.NewRequest("POST", "/api/v2/spans",
		bytes.NewBuffer([]byte(body)))
	r.Header.Add("content-type", "application/json")

	next := &zipkinMockTraceConsumer{
		ch: make(chan pdata.Traces, 10),
	}
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: zipkinReceiverName,
		},
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: "",
		},
	}
	zr, err := New(cfg, next)
	require.NoError(t, err)

	req := httptest.NewRecorder()
	zr.ServeHTTP(req, r)

	require.Equal(t, 400, req.Code)
	require.Equal(t, "invalid character 'i' looking for beginning of object key string\n", req.Body.String())
}

func TestReceiverConsumerError(t *testing.T) {
	body, err := ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v2_single.json")
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/api/v2/spans", bytes.NewBuffer(body))
	r.Header.Add("content-type", "application/json")

	next := &zipkinMockTraceConsumer{
		ch:  make(chan pdata.Traces, 10),
		err: errors.New("consumer error"),
	}
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: zipkinReceiverName,
		},
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: "localhost:9411",
		},
	}
	zr, err := New(cfg, next)
	require.NoError(t, err)

	req := httptest.NewRecorder()
	zr.ServeHTTP(req, r)

	require.Equal(t, 500, req.Code)
	require.Equal(t, "\"Internal Server Error\"", req.Body.String())
}

func thriftExample() []byte {
	now := time.Now().Unix()
	zSpans := []*zipkincore.Span{
		{
			TraceID: 1,
			Name:    "test",
			ID:      2,
			BinaryAnnotations: []*zipkincore.BinaryAnnotation{
				{
					Key:   "http.path",
					Value: []byte("/"),
				},
			},
			Timestamp: &now,
		},
	}

	return zipkin2.SerializeThrift(zSpans)
}

func compressGzip(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(body)
	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

func compressZlib(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)

	_, err := zw.Write(body)
	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

type zipkinMockTraceConsumer struct {
	ch  chan pdata.Traces
	err error
}

func (m *zipkinMockTraceConsumer) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	m.ch <- td
	return m.err
}

func TestConvertSpansToTraceSpans_JSONWithoutSerivceName(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/sample2.json")
	require.NoError(t, err, "Failed to read sample JSON file: %v", err)
	zi := new(ZipkinReceiver)
	zi.config = createDefaultConfig().(*Config)
	reqs, err := zi.v2ToTraceSpans(blob, nil)
	require.NoError(t, err, "Failed to parse convert Zipkin spans in JSON to Trace spans: %v", err)

	require.Equal(t, 1, reqs.ResourceSpans().Len(), "Expecting only one request since all spans share same node/localEndpoint: %v", reqs.ResourceSpans().Len())

	// Expecting 1 non-nil spans
	require.Equal(t, 1, reqs.SpanCount(), "Incorrect non-nil spans count")
}

func TestReceiverConvertsStringsToTypes(t *testing.T) {
	body, err := ioutil.ReadFile("../../translator/trace/zipkin/testdata/zipkin_v2_single.json")
	require.NoError(t, err, "Failed to read sample JSON file: %v", err)

	r := httptest.NewRequest("POST", "/api/v2/spans", bytes.NewBuffer(body))
	r.Header.Add("content-type", "application/json")

	next := &zipkinMockTraceConsumer{
		ch: make(chan pdata.Traces, 10),
	}
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: zipkinReceiverName,
		},
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: "",
		},
		ParseStringTags: true,
	}
	zr, err := New(cfg, next)
	require.NoError(t, err)

	req := httptest.NewRecorder()
	zr.ServeHTTP(req, r)

	select {
	case td := <-next.ch:
		require.NotNil(t, td)
		require.Equal(t, 202, req.Code)

		span := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)

		expected := pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
			"cache_hit":            pdata.NewAttributeValueBool(true),
			"ping_count":           pdata.NewAttributeValueInt(25),
			"timeout":              pdata.NewAttributeValueDouble(12.3),
			"clnt/finagle.version": pdata.NewAttributeValueString("6.45.0"),
			"http.path":            pdata.NewAttributeValueString("/api"),
			"http.status_code":     pdata.NewAttributeValueInt(500),
			"net.host.ip":          pdata.NewAttributeValueString("7::80:807f"),
			"peer.service":         pdata.NewAttributeValueString("backend"),
			"net.peer.ip":          pdata.NewAttributeValueString("192.168.99.101"),
			"net.peer.port":        pdata.NewAttributeValueInt(9000),
		}).Sort()

		actual := span.Attributes().Sort()

		assert.EqualValues(t, expected, actual)
		break
	case <-time.After(time.Second * 2):
		t.Error("next consumer did not receive the batch")
		break
	}
}
