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

package zipkinexporter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
)

var translator zipkinv2.FromTranslator

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to receive traffic from
// Zipkin servers and then transform them back to the final form when creating an
// OpenCensus spandata.
type zipkinExporter struct {
	defaultServiceName string

	url            string
	client         *http.Client
	serializer     zipkinreporter.SpanSerializer
	clientSettings *confighttp.HTTPClientSettings
}

func createZipkinExporter(cfg *Config) (*zipkinExporter, error) {
	ze := &zipkinExporter{
		defaultServiceName: cfg.DefaultServiceName,
		url:                cfg.Endpoint,
		clientSettings:     &cfg.HTTPClientSettings,
		client:             nil,
	}

	switch cfg.Format {
	case "json":
		ze.serializer = zipkinreporter.JSONSerializer{}
	case "proto":
		ze.serializer = zipkin_proto3.SpanSerializer{}
	default:
		return nil, fmt.Errorf("%s is not one of json or proto", cfg.Format)
	}

	return ze, nil
}

// start creates the http client
func (ze *zipkinExporter) start(_ context.Context, host component.Host) (err error) {
	ze.client, err = ze.clientSettings.ToClient(host.GetExtensions())
	return
}

func (ze *zipkinExporter) pushTraces(ctx context.Context, td pdata.Traces) error {
	spans, err := translator.FromTraces(td)
	if err != nil {
		return consumererror.Permanent(fmt.Errorf("failed to push trace data via Zipkin exporter: %w", err))
	}

	body, err := ze.serializer.Serialize(spans)
	if err != nil {
		return consumererror.Permanent(fmt.Errorf("failed to push trace data via Zipkin exporter: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ze.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to push trace data via Zipkin exporter: %w", err)
	}
	req.Header.Set("Content-Type", ze.serializer.ContentType())

	resp, err := ze.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push trace data via Zipkin exporter: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("failed the request with status code %d", resp.StatusCode)
	}
	return nil
}
