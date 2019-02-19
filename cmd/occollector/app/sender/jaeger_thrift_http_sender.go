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

package sender

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"go.uber.org/zap"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	jaegertranslator "github.com/census-instrumentation/opencensus-service/translator/trace/jaeger"
)

// Default timeout for http request in seconds
const defaultHTTPTimeout = time.Second * 5

// JaegerThriftHTTPSender forwards spans encoded in the jaeger thrift
// format to a http server
type JaegerThriftHTTPSender struct {
	url     string
	headers map[string]string
	client  *http.Client
	logger  *zap.Logger
}

// HTTPOption sets a parameter for the HttpCollector
type HTTPOption func(s *JaegerThriftHTTPSender)

// HTTPTimeout sets maximum timeout for http request.
func HTTPTimeout(duration time.Duration) HTTPOption {
	return func(s *JaegerThriftHTTPSender) { s.client.Timeout = duration }
}

// HTTPRoundTripper configures the underlying Transport on the *http.Client
// that is used
func HTTPRoundTripper(transport http.RoundTripper) HTTPOption {
	return func(s *JaegerThriftHTTPSender) {
		s.client.Transport = transport
	}
}

// NewJaegerThriftHTTPSender returns a new HTTP-backend span sender. url should be an http
// url of the collector to handle POST request, typically something like:
//     http://hostname:14268/api/traces?format=jaeger.thrift
func NewJaegerThriftHTTPSender(
	url string,
	headers map[string]string,
	zlogger *zap.Logger,
	options ...HTTPOption,
) *JaegerThriftHTTPSender {
	s := &JaegerThriftHTTPSender{
		url:     url,
		headers: headers,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
		logger:  zlogger,
	}

	for _, option := range options {
		option(s)
	}
	return s
}

// ProcessSpans sends the received data to the configured Jaeger Thrift end-point.
func (s *JaegerThriftHTTPSender) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	// TODO: (@pjanotti) In case of failure the translation to Jaeger Thrift is going to be remade, cache it somehow.
	if batch == nil {
		return 0, fmt.Errorf("Jaeger sender received nil batch")
	}

	tBatch, err := jaegertranslator.OCProtoToJaegerThrift(batch)
	if err != nil {
		return uint64(len(batch.Spans)), err
	}

	mSpans := tBatch.Spans
	body, err := serializeThrift(tBatch)
	if err != nil {
		return uint64(len(mSpans)), err
	}
	req, err := http.NewRequest("POST", s.url, body)
	if err != nil {
		return uint64(len(mSpans)), err
	}
	req.Header.Set("Content-Type", "application/x-thrift")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return uint64(len(mSpans)), err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return uint64(len(mSpans)), fmt.Errorf("Jaeger Thirft HTTP sender error: %d", resp.StatusCode)
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
