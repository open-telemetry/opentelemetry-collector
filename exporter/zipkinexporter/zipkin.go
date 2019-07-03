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

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/spf13/viper"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/errors/errorkind"
	"github.com/open-telemetry/opentelemetry-service/observability"
	"github.com/open-telemetry/opentelemetry-service/translator/trace"
	spandatatranslator "github.com/open-telemetry/opentelemetry-service/translator/trace/spandata"
)

// ZipkinConfig holds the configuration of a Zipkin exporter.
type ZipkinConfig struct {
	ServiceName      string         `mapstructure:"service_name,omitempty"`
	Endpoint         string         `mapstructure:"endpoint,omitempty"`
	LocalEndpointURI string         `mapstructure:"local_endpoint,omitempty"`
	UploadPeriod     *time.Duration `mapstructure:"upload_period,omitempty"`
}

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to receive traffic from
// Zipkin servers and then transform them back to the final form when creating an
// OpenCensus spandata.
type zipkinExporter struct {
	// mu protects the fields below
	mu sync.Mutex

	defaultServiceName      string
	defaultLocalEndpointURI string

	endpointURI string
	reporter    zipkinreporter.Reporter
}

// Default values for Zipkin endpoint.
const (
	DefaultZipkinEndpointHostPort = "localhost:9411"
	DefaultZipkinEndpointURL      = "http://" + DefaultZipkinEndpointHostPort + "/api/v2/spans"
)

// EndpointURL returns the endpoint URL of the Zipkin configuration.
func (zc *ZipkinConfig) EndpointURL() string {
	// If no endpoint was set, use the default Zipkin reporter URI.
	endpoint := DefaultZipkinEndpointURL
	if zc != nil && zc.Endpoint != "" {
		endpoint = zc.Endpoint
	}
	return endpoint
}

// ZipkinExportersFromViper unmarshals the viper and returns an exporter.TraceExporter targeting
// Zipkin according to the configuration settings.
func ZipkinExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
	var cfg struct {
		Zipkin *ZipkinConfig `mapstructure:"zipkin"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}

	zc := cfg.Zipkin
	if zc == nil {
		return nil, nil, nil, nil
	}

	serviceName := ""
	if zc.ServiceName != "" {
		serviceName = zc.ServiceName
	}
	localEndpointURI := "192.168.1.5:5454"
	if zc.LocalEndpointURI != "" {
		localEndpointURI = zc.LocalEndpointURI
	}
	endpoint := zc.EndpointURL()
	var uploadPeriod time.Duration
	if zc.UploadPeriod != nil && *zc.UploadPeriod > 0 {
		uploadPeriod = *zc.UploadPeriod
	}
	zle, err := newZipkinExporter(endpoint, serviceName, localEndpointURI, uploadPeriod)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Cannot configure Zipkin exporter: %v", err)
	}
	tps = append(tps, zle)
	doneFns = append(doneFns, zle.stop)
	return
}

func newZipkinExporter(finalEndpointURI, defaultServiceName, defaultLocalEndpointURI string, uploadPeriod time.Duration) (*zipkinExporter, error) {
	var opts []zipkinhttp.ReporterOption
	if uploadPeriod > 0 {
		opts = append(opts, zipkinhttp.BatchInterval(uploadPeriod))
	}
	reporter := zipkinhttp.NewReporter(finalEndpointURI, opts...)
	zle := &zipkinExporter{
		endpointURI:             finalEndpointURI,
		defaultServiceName:      defaultServiceName,
		defaultLocalEndpointURI: defaultLocalEndpointURI,
		reporter:                reporter,
	}
	return zle, nil
}

func lookupAttribute(node *commonpb.Node, key string) string {
	if node == nil {
		return ""
	}
	return node.Attributes[key]
}

func zipkinEndpointFromNode(node *commonpb.Node, serviceName string, endpointType zipkinDirection) *zipkinmodel.Endpoint {
	if node == nil {
		return nil
	}

	// The data in the Attributes map was saved in the format
	// {
	//      "ipv4": "192.168.99.101",
	//      "port": "9000",
	//      "serviceName": "backend",
	// }
	attributes := node.Attributes

	var ipv4Key, ipv6Key, portKey string
	if endpointType == isLocalEndpoint {
		ipv4Key, ipv6Key, portKey = "ipv4", "ipv6", "port"
	} else {
		ipv4Key, ipv6Key, portKey = "zipkin.remoteEndpoint.ipv4", "zipkin.remoteEndpoint.ipv6", "zipkin.remoteEndpoint.port"
	}

	var ip net.IP
	ipv6Selected := false
	if ipv4 := attributes[ipv4Key]; ipv4 != "" {
		ip = net.ParseIP(ipv4)
	} else if ipv6 := attributes[ipv6Key]; ipv6 != "" {
		ip = net.ParseIP(ipv6)
		ipv6Selected = true
	}

	port, _ := strconv.ParseUint(attributes[portKey], 10, 16)
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

func (ze *zipkinExporter) stop() error {
	ze.mu.Lock()
	defer ze.mu.Unlock()

	return ze.reporter.Close()
}

func (ze *zipkinExporter) ConsumeTraceData(ctx context.Context, td data.TraceData) (zerr error) {
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
			return errorkind.Permanent(err)
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
	observability.RecordTraceExporterMetrics(observability.ContextWithExporterName(ctx, "zipkin"), len(td.Spans), len(td.Spans)-goodSpans)

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

const zipkinRemoteEndpointKey = "zipkin.remoteEndpoint.serviceName"

func (ze *zipkinExporter) zipkinSpan(node *commonpb.Node, s *trace.SpanData) (zc zipkinmodel.SpanModel) {
	localEndpointServiceName := ze.serviceNameOrDefault(node)
	localEndpoint := zipkinEndpointFromNode(node, localEndpointServiceName, isLocalEndpoint)

	remoteServiceName := lookupAttribute(node, zipkinRemoteEndpointKey)
	remoteEndpoint := zipkinEndpointFromNode(node, remoteServiceName, isRemoteEndpoint)

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
