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

package exporterparser

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/trace"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type ZipkinConfig struct {
	ServiceName      string         `yaml:"service_name,omitempty"`
	Endpoint         string         `yaml:"endpoint,omitempty"`
	LocalEndpointURI string         `yaml:"local_endpoint,omitempty"`
	UploadPeriod     *time.Duration `yaml:"upload_period,omitempty"`
}

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to intercept traffic from
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

const (
	DefaultZipkinEndpointHostPort = "localhost:9411"
	DefaultZipkinEndpointURL      = "http://" + DefaultZipkinEndpointHostPort + "/api/v2/spans"
)

func (zc *ZipkinConfig) EndpointURL() string {
	// If no endpoint was set, use the default Zipkin reporter URI.
	endpoint := DefaultZipkinEndpointURL
	if zc != nil && zc.Endpoint != "" {
		endpoint = zc.Endpoint
	}
	return endpoint
}

func ZipkinExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var cfg struct {
		Exporters *struct {
			Zipkin *ZipkinConfig `yaml:"zipkin"`
		} `yaml:"exporters"`
	}
	if err := yamlUnmarshal(config, &cfg); err != nil {
		return nil, nil, err
	}
	if cfg.Exporters == nil {
		return nil, nil, nil
	}

	zc := cfg.Exporters.Zipkin
	if zc == nil {
		return nil, nil, nil
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
		return nil, nil, fmt.Errorf("Cannot configure Zipkin exporter: %v", err)
	}
	tes = append(tes, zle)
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

func zipkinEndpointFromNode(node *commonpb.Node, serviceName string, endpointType zipkinDirection) (*zipkinmodel.Endpoint, error) {
	if node == nil {
		return nil, nil
	}

	// The data in the Attributes map was saved in the format
	// {
	//      "ipv4": "192.168.99.101",
	//      "port": "9000",
	//      "serviceName": "backend",
	// }
	attributes := node.Attributes

	var endpointURI string

	ipv6Selected := false
	if ipv4 := attributes[endpointType.ipv4Key()]; ipv4 != "" {
		endpointURI = ipv4
	} else if ipv6 := attributes[endpointType.ipv6Key()]; ipv6 != "" {
		// In this case for ipv6 address when combining with a port,
		// we need them enclosed in square brackets as per
		//    https://golang.org/pkg/net/#SplitHostPort
		endpointURI = fmt.Sprintf("[%s]", ipv6)
		ipv6Selected = true
	}

	port := attributes[endpointType.portKey()]
	if port != "" {
		endpointURI += ":" + port
	} else if ipv6Selected { // net.SplitHostPort requires a port of ipv6 values
		endpointURI += ":" + "0"
	}
	return openzipkin.NewEndpoint(serviceName, endpointURI)
}

func (ze *zipkinExporter) stop() error {
	ze.mu.Lock()
	defer ze.mu.Unlock()

	return ze.reporter.Close()
}

func (ze *zipkinExporter) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	for _, sd := range spandata {
		zs, err := ze.zipkinSpan(node, sd)
		if err == nil {
			// ze.reporter can get closed in the midst of a Send
			// so avoid a read/write during that mutation.
			ze.mu.Lock()
			ze.reporter.Send(zs)
			ze.mu.Unlock()
		}
	}
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
	return zipkinmodel.TraceID{
		High: binary.BigEndian.Uint64(t[:8]),
		Low:  binary.BigEndian.Uint64(t[8:]),
	}
}

func convertSpanID(s trace.SpanID) zipkinmodel.ID {
	return zipkinmodel.ID(binary.BigEndian.Uint64(s[:]))
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

func (d zipkinDirection) ipv6Key() string {
	if d == isLocalEndpoint {
		return "ipv6"
	}
	return "zipkin.remoteEndpoint.ipv6"
}

func (d zipkinDirection) portKey() string {
	if d == isLocalEndpoint {
		return "port"
	}
	return "zipkin.remoteEndpoint.port"
}

func (d zipkinDirection) ipv4Key() string {
	if d == isLocalEndpoint {
		return "ipv4"
	}
	return "zipkin.remoteEndpoint.ipv4"
}

const zipkinRemoteEndpointKey = "zipkin.remoteEndpoint.serviceName"

func (ze *zipkinExporter) zipkinSpan(node *commonpb.Node, s *trace.SpanData) (zc zipkinmodel.SpanModel, err error) {
	localEndpointServiceName := ze.serviceNameOrDefault(node)
	localEndpoint, err := zipkinEndpointFromNode(node, localEndpointServiceName, isLocalEndpoint)
	if err != nil {
		return zc, err
	}

	remoteServiceName := node.Attributes[zipkinRemoteEndpointKey]
	remoteEndpoint, _ := zipkinEndpointFromNode(node, remoteServiceName, isRemoteEndpoint)

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

	return z, nil
}
