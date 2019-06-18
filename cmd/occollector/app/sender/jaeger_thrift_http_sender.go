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

package sender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/data"
	jaegertranslator "github.com/open-telemetry/opentelemetry-service/translator/trace/jaeger"
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

var _ consumer.TraceConsumer = (*JaegerThriftHTTPSender)(nil)

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

// ConsumeTraceData sends the received data to the configured Jaeger Thrift end-point.
func (s *JaegerThriftHTTPSender) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	// TODO: (@pjanotti) In case of failure the translation to Jaeger Thrift is going to be remade, cache it somehow.
	tBatch, err := jaegertranslator.OCProtoToJaegerThrift(td)
	if err != nil {
		return err
	}

	body, err := serializeThrift(tBatch)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-thrift")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("Jaeger Thirft HTTP sender error: %d", resp.StatusCode)
	}
	return nil
}

func serializeThrift(obj thrift.TStruct) (*bytes.Buffer, error) {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	if err := obj.Write(p); err != nil {
		return nil, err
	}
	return t.Buffer, nil
}
