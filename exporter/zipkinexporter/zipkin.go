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
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinproto "github.com/openzipkin/zipkin-go/proto/v2"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
	spandatatranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/spandata"
	"github.com/open-telemetry/opentelemetry-collector/translator/trace/zipkin"
)

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to receive traffic from
// Zipkin servers and then transform them back to the final form when creating an
// OpenCensus spandata.
type zipkinExporter struct {
	// mu protects the fields below
	mu sync.Mutex

	defaultServiceName string

	reporter zipkinreporter.Reporter
}

// Default values for Zipkin endpoint.
const (
	DefaultZipkinEndpointHostPort = "localhost:9411"
	DefaultZipkinEndpointURL      = "http://" + DefaultZipkinEndpointHostPort + "/api/v2/spans"
)

func newZipkinExporter(finalEndpointURI, defaultServiceName string, uploadPeriod time.Duration, format string) (*zipkinExporter, error) {
	var opts []zipkinhttp.ReporterOption
	if uploadPeriod > 0 {
		opts = append(opts, zipkinhttp.BatchInterval(uploadPeriod))
	}
	// default is json
	switch format {
	case "json":
		break
	case "proto":
		opts = append(opts, zipkinhttp.Serializer(zipkinproto.SpanSerializer{}))
	default:
		return nil, fmt.Errorf("%s is not one of json or proto", format)
	}
	reporter := zipkinhttp.NewReporter(finalEndpointURI, opts...)
	zle := &zipkinExporter{
		defaultServiceName: defaultServiceName,
		reporter:           reporter,
	}
	return zle, nil
}

// zipkinEndpointFromAttributes extracts zipkin endpoint information
// from a set of attributes (in the format of OC SpanData). It returns the built
// zipkin endpoint and insert the attribute keys that were made redundant
// (because now they can be represented by the endpoint) into the redundantKeys
// map (function assumes that this was created by the caller). Per call at most
// 3 attribute keys can be made redundant.
func zipkinEndpointFromAttributes(
	attributes map[string]interface{},
	serviceName string,
	endpointType zipkinDirection,
	redundantKeys map[string]bool,
) (endpoint *zipkinmodel.Endpoint) {

	if attributes == nil {
		return nil
	}

	// The data in the Attributes map was saved in the format
	// {
	//      "ipv4": "192.168.99.101",
	//      "port": "9000",
	//      "serviceName": "backend",
	// }

	var ipv4Key, ipv6Key, portKey string
	if endpointType == isLocalEndpoint {
		ipv4Key, ipv6Key, portKey = zipkin.LocalEndpointIPv4, zipkin.LocalEndpointIPv6, zipkin.LocalEndpointPort
	} else {
		ipv4Key, ipv6Key, portKey = zipkin.RemoteEndpointIPv4, zipkin.RemoteEndpointIPv6, zipkin.RemoteEndpointPort
	}

	var ip net.IP
	ipv6Selected := false

	if ipv4Str, ok := extractStringAttribute(attributes, ipv4Key); ok {
		ip = net.ParseIP(ipv4Str)
		redundantKeys[ipv4Key] = true
	} else if ipv6Str, ok := extractStringAttribute(attributes, ipv6Key); ok {
		ip = net.ParseIP(ipv6Str)
		ipv6Selected = true
		redundantKeys[ipv6Key] = true
	}

	var port uint64
	if portStr, ok := extractStringAttribute(attributes, portKey); ok {
		port, _ = strconv.ParseUint(portStr, 10, 16)
		redundantKeys[portKey] = true
	}

	if serviceName == "" && len(ip) == 0 && port == 0 {
		// Nothing to put on the endpoint
		return nil
	}

	zEndpoint := &zipkinmodel.Endpoint{
		ServiceName: serviceName,
		Port:        uint16(port),
	}

	if ipv6Selected {
		zEndpoint.IPv6 = ip
	} else {
		zEndpoint.IPv4 = ip
	}

	return zEndpoint
}

func extractStringAttribute(
	attributes map[string]interface{},
	key string,
) (value string, ok bool) {
	var i interface{}
	if i, ok = attributes[key]; ok {
		value, ok = i.(string)
	}

	return value, ok
}

func (ze *zipkinExporter) Start(host component.Host) error {
	return nil
}

func (ze *zipkinExporter) Shutdown() error {
	ze.mu.Lock()
	defer ze.mu.Unlock()

	return ze.reporter.Close()
}

func (ze *zipkinExporter) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) (zerr error) {
	ctx, span := trace.StartSpan(ctx,
		"opencensus.service.exporter.zipkin.ExportTrace",
		trace.WithSampler(trace.NeverSample()))

	defer func() {
		if zerr != nil && span.IsRecordingEvents() {
			span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: zerr.Error()})
		}
		span.End()
	}()

	goodSpans := 0
	for _, span := range td.Spans {
		sd, err := spandatatranslator.ProtoSpanToOCSpanData(span)
		if err != nil {
			return consumererror.Permanent(err)
		}
		zs := ze.zipkinSpan(td.Node, sd)
		// ze.reporter can get closed in the midst of a Send
		// so avoid a read/write during that mutation.
		ze.mu.Lock()
		ze.reporter.Send(zs)
		ze.mu.Unlock()
		goodSpans++
	}

	// And finally record metrics on the number of exported spans.
	observability.RecordMetricsForTraceExporter(observability.ContextWithExporterName(ctx, "zipkin"), len(td.Spans), len(td.Spans)-goodSpans)

	return nil
}

// This code from down below is mostly copied from
// https://github.com/census-instrumentation/opencensus-go/blob/96e75b88df843315da521168a0e3b11792088728/exporter/zipkin/zipkin.go#L57-L194
// but that is because the Zipkin Go exporter requires process to change
// and was designed without taking into account that LocalEndpoint and RemoteEndpoint
// are per-span-Node attributes instead of global/system variables.
// The alternative is to create a single exporter for every single combination
// but this wastes resources i.e. an HTTP client for every single combination
// but also requires the exporter to be changed entirely as per
// https://github.com/census-instrumentation/opencensus-go/issues/959
//
// TODO: (@odeke-em) whenever we come to consensus with the OpenCensus-Go repository
// on the Zipkin exporter and they have the same logic, then delete all the code
// below here to allow per-span configuration changes.

const (
	statusCodeTagKey        = "error"
	statusDescriptionTagKey = "opencensus.status_description"
)

var (
	sampledTrue    = true
	canonicalCodes = [...]string{
		"OK",
		"CANCELLED",
		"UNKNOWN",
		"INVALID_ARGUMENT",
		"DEADLINE_EXCEEDED",
		"NOT_FOUND",
		"ALREADY_EXISTS",
		"PERMISSION_DENIED",
		"RESOURCE_EXHAUSTED",
		"FAILED_PRECONDITION",
		"ABORTED",
		"OUT_OF_RANGE",
		"UNIMPLEMENTED",
		"INTERNAL",
		"UNAVAILABLE",
		"DATA_LOSS",
		"UNAUTHENTICATED",
	}
)

func canonicalCodeString(code int32) string {
	if code < 0 || int(code) >= len(canonicalCodes) {
		return "error code " + strconv.FormatInt(int64(code), 10)
	}
	return canonicalCodes[code]
}

func convertTraceID(t trace.TraceID) zipkinmodel.TraceID {
	h, l, _ := tracetranslator.BytesToUInt64TraceID(t[:])
	return zipkinmodel.TraceID{High: h, Low: l}
}

func convertSpanID(s trace.SpanID) zipkinmodel.ID {
	id, _ := tracetranslator.BytesToUInt64SpanID(s[:])
	return zipkinmodel.ID(id)
}

func spanKind(s *trace.SpanData) zipkinmodel.Kind {
	switch s.SpanKind {
	case trace.SpanKindClient:
		return zipkinmodel.Client
	case trace.SpanKindServer:
		return zipkinmodel.Server
	}
	return zipkinmodel.Undetermined
}

func (ze *zipkinExporter) serviceNameOrDefault(node *commonpb.Node) string {
	// ze.defaultServiceName should never change
	defaultServiceName := ze.defaultServiceName

	if node == nil || node.ServiceInfo == nil {
		return defaultServiceName
	}

	if node.ServiceInfo.Name == "" {
		return defaultServiceName
	}

	return node.ServiceInfo.Name
}

type zipkinDirection bool

const (
	isLocalEndpoint  zipkinDirection = true
	isRemoteEndpoint zipkinDirection = false
)

func (ze *zipkinExporter) zipkinSpan(
	node *commonpb.Node,
	s *trace.SpanData,
) (zc zipkinmodel.SpanModel) {

	// Per call to zipkinEndpointFromAttributes at most 3 attribute keys can be
	// made redundant, give a hint when calling make that we expect at most 6
	// items on the map.
	redundantKeys := make(map[string]bool, 6)
	localEndpointServiceName := ze.serviceNameOrDefault(node)
	localEndpoint := zipkinEndpointFromAttributes(
		s.Attributes, localEndpointServiceName, isLocalEndpoint, redundantKeys)

	remoteServiceName := ""
	if remoteServiceEntry, ok := s.Attributes[zipkin.RemoteEndpointServiceName]; ok {
		if remoteServiceName, ok = remoteServiceEntry.(string); ok {
			redundantKeys[zipkin.RemoteEndpointServiceName] = true
		}
	}
	remoteEndpoint := zipkinEndpointFromAttributes(
		s.Attributes, remoteServiceName, isRemoteEndpoint, redundantKeys)

	sc := s.SpanContext
	z := zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID: convertTraceID(sc.TraceID),
			ID:      convertSpanID(sc.SpanID),
			Sampled: &sampledTrue,
		},
		Kind:           spanKind(s),
		Name:           s.Name,
		Timestamp:      s.StartTime,
		Shared:         false,
		LocalEndpoint:  localEndpoint,
		RemoteEndpoint: remoteEndpoint,
	}

	if s.ParentSpanID != (trace.SpanID{}) {
		id := convertSpanID(s.ParentSpanID)
		z.ParentID = &id
	}

	if s, e := s.StartTime, s.EndTime; !s.IsZero() && !e.IsZero() {
		z.Duration = e.Sub(s)
	}

	// construct Tags from s.Attributes and s.Status.
	if len(s.Attributes) != 0 {
		m := make(map[string]string, len(s.Attributes)+2)
		for key, value := range s.Attributes {
			if redundantKeys[key] {
				// Already represented by something other than an attribute,
				// skip it.
				continue
			}

			switch v := value.(type) {
			case string:
				m[key] = v
			case bool:
				if v {
					m[key] = "true"
				} else {
					m[key] = "false"
				}
			case int64:
				m[key] = strconv.FormatInt(v, 10)
			}
		}
		z.Tags = m
	}
	if s.Status.Code != 0 || s.Status.Message != "" {
		if z.Tags == nil {
			z.Tags = make(map[string]string, 2)
		}
		if s.Status.Code != 0 {
			z.Tags[statusCodeTagKey] = canonicalCodeString(s.Status.Code)
		}
		if s.Status.Message != "" {
			z.Tags[statusDescriptionTagKey] = s.Status.Message
		}
	}

	// construct Annotations from s.Annotations and s.MessageEvents.
	if len(s.Annotations) != 0 || len(s.MessageEvents) != 0 {
		z.Annotations = make([]zipkinmodel.Annotation, 0, len(s.Annotations)+len(s.MessageEvents))
		for _, a := range s.Annotations {
			z.Annotations = append(z.Annotations, zipkinmodel.Annotation{
				Timestamp: a.Time,
				Value:     a.Message,
			})
		}
		for _, m := range s.MessageEvents {
			a := zipkinmodel.Annotation{
				Timestamp: m.Time,
			}
			switch m.EventType {
			case trace.MessageEventTypeSent:
				a.Value = "SENT"
			case trace.MessageEventTypeRecv:
				a.Value = "RECV"
			default:
				a.Value = "<?>"
			}
			z.Annotations = append(z.Annotations, a)
		}
	}

	return z
}
