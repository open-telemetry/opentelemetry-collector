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

package jaegerthrifthttpexporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/jaeger"
)

// Default timeout for http request in seconds
const defaultHTTPTimeout = time.Second * 5

// New returns a new Jaeger Thrift over HTTP exporter.
// The exporterName is the name to be used in the observability of the exporter.
// The httpAddress should be the URL of the collector to handle POST requests,
// typically something like: http://hostname:14268/api/traces.
// The headers parameter is used to add entries to the POST message set to the
// collector.
// The timeout is used to set the timeout for the HTTP requests, if the
// value is equal or smaller than zero the default of 5 seconds is used.
func New(
	config configmodels.Exporter,
	httpAddress string,
	headers map[string]string,
	timeout time.Duration,
) (exporter.TraceExporter, error) {

	clientTimeout := defaultHTTPTimeout
	if timeout != 0 {
		clientTimeout = timeout
	}
	s := &jaegerThriftHTTPSender{
		url:     httpAddress,
		headers: headers,
		client:  &http.Client{Timeout: clientTimeout},
	}

	exp, err := exporterhelper.NewTraceExporter(
		config,
		s.pushTraceData,
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true))

	return exp, err
}

// jaegerThriftHTTPSender forwards spans encoded in the jaeger thrift
// format to a http server.
type jaegerThriftHTTPSender struct {
	url     string
	headers map[string]string
	client  *http.Client
}

func (s *jaegerThriftHTTPSender) pushTraceData(
	ctx context.Context,
	td consumerdata.TraceData,
) (droppedSpans int, err error) {

	tBatch, err := jaegertranslator.OCProtoToJaegerThrift(td)
	if err != nil {
		return len(td.Spans), consumererror.Permanent(err)
	}

	body, err := serializeThrift(tBatch)
	if err != nil {
		return len(td.Spans), err
	}

	req, err := http.NewRequest("POST", s.url, body)
	if err != nil {
		return len(td.Spans), err
	}

	req.Header.Set("Content-Type", "application/x-thrift")
	if s.headers != nil {
		for k, v := range s.headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return len(td.Spans), err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		err = fmt.Errorf(
			"HTTP %d %q",
			resp.StatusCode,
			http.StatusText(resp.StatusCode))
		return len(td.Spans), err
	}

	return 0, nil
}

func serializeThrift(obj thrift.TStruct) (*bytes.Buffer, error) {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	if err := obj.Write(p); err != nil {
		return nil, err
	}
	return t.Buffer, nil
}
