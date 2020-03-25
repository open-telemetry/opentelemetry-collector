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
	"net/http"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinproto "github.com/openzipkin/zipkin-go/proto/v2"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	spandatatranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/spandata"
	"github.com/open-telemetry/opentelemetry-collector/translator/trace/zipkin"
)

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to receive traffic from
// Zipkin servers and then transform them back to the final form when creating an
// OpenCensus spandata.
type zipkinExporter struct {
	defaultServiceName string

	exportResourceLabels bool
	url                  string
	client               *http.Client
	serializer           zipkinreporter.SpanSerializer
}

// Default values for Zipkin endpoint.
const (
	defaultTimeout = time.Second * 5

	defaultServiceName string = "<missing service name>"

	DefaultExportResourceLabels   = true
	DefaultZipkinEndpointHostPort = "localhost:9411"
	DefaultZipkinEndpointURL      = "http://" + DefaultZipkinEndpointHostPort + "/api/v2/spans"
)

// NewTraceExporter creates an zipkin trace exporter.
func NewTraceExporter(logger *zap.Logger, config configmodels.Exporter) (component.TraceExporterOld, error) {
	ze, err := createZipkinExporter(logger, config)
	if err != nil {
		return nil, err
	}
	zexp, err := exporterhelper.NewTraceExporter(
		config,
		ze.PushTraceData)
	if err != nil {
		return nil, err
	}

	return zexp, nil
}

func createZipkinExporter(logger *zap.Logger, config configmodels.Exporter) (*zipkinExporter, error) {
	zCfg := config.(*Config)

	serviceName := defaultServiceName
	if zCfg.DefaultServiceName != "" {
		serviceName = zCfg.DefaultServiceName
	}

	exportResourceLabels := DefaultExportResourceLabels
	if zCfg.ExportResourceLabels != nil {
		exportResourceLabels = *zCfg.ExportResourceLabels
	}

	ze := &zipkinExporter{
		defaultServiceName:   serviceName,
		exportResourceLabels: exportResourceLabels,
		url:                  zCfg.URL,
		client:               &http.Client{Timeout: defaultTimeout},
	}

	switch zCfg.Format {
	case "json":
		ze.serializer = zipkinreporter.JSONSerializer{}
	case "proto":
		ze.serializer = zipkinproto.SpanSerializer{}
	default:
		return nil, fmt.Errorf("%s is not one of json or proto", zCfg.Format)
	}

	return ze, nil
}

func (ze *zipkinExporter) PushTraceData(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error) {
	tbatch := []*zipkinmodel.SpanModel{}

	var resource *resourcepb.Resource
	if ze.exportResourceLabels {
		resource = td.Resource
	}

	for _, span := range td.Spans {
		sd, err := spandatatranslator.ProtoSpanToOCSpanData(span, resource)
		if err != nil {
			return len(td.Spans), consumererror.Permanent(err)
		}
		zs := ze.zipkinSpan(td.Node, sd)
		tbatch = append(tbatch, &zs)
	}

	body, err := ze.serializer.Serialize(tbatch)
	if err != nil {
		return len(td.Spans), err
	}

	req, err := http.NewRequest("POST", ze.url, bytes.NewReader(body))
	if err != nil {
		return len(td.Spans), err
	}
	req.Header.Set("Content-Type", ze.serializer.ContentType())

	resp, err := ze.client.Do(req)
	if err != nil {
		return len(td.Spans), err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return len(td.Spans), fmt.Errorf("failed the request with status code %d", resp.StatusCode)
	}
	return 0, nil
}

func (ze *zipkinExporter) zipkinSpan(
	node *commonpb.Node,
	s *trace.SpanData,
) (zc zipkinmodel.SpanModel) {
	return zipkin.OCSpanDataToZipkin(node, s, ze.defaultServiceName)
}
