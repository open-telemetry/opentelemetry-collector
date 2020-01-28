// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkin

import (
	"net"
	"strconv"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"go.opencensus.io/trace"

	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

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
		ipv4Key, ipv6Key, portKey = LocalEndpointIPv4, LocalEndpointIPv6, LocalEndpointPort
	} else {
		ipv4Key, ipv6Key, portKey = RemoteEndpointIPv4, RemoteEndpointIPv6, RemoteEndpointPort
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

func convertTraceID(t trace.TraceID) zipkinmodel.TraceID {
	h, l, _ := tracetranslator.BytesToUInt64TraceID(t[:])
	return zipkinmodel.TraceID{High: h, Low: l}
}

func convertSpanID(s trace.SpanID) zipkinmodel.ID {
	id, _ := tracetranslator.BytesToUInt64SpanID(s[:])
	return zipkinmodel.ID(id)
}

func spanKind(s *trace.SpanData) zipkinmodel.Kind {
	// First try span kinds that are set directly in SpanKind field.
	switch s.SpanKind {
	case trace.SpanKindClient:
		return zipkinmodel.Client
	case trace.SpanKindServer:
		return zipkinmodel.Server
	}

	// SpanKind==SpanKindUnspecified, check if TagSpanKind attribute is set.
	// This can happen if span kind had no equivalent in OC, so we could represent it in
	// the SpanKind. In that case we had set a special attribute TagSpanKind in
	// the span to preserve the span kind. Now we will do a reverse translation.
	spanKind := s.Attributes[tracetranslator.TagSpanKind]
	if spanKind != nil {
		// Yes the attribute is present. Check that it is a string.
		spanKindStr, ok := spanKind.(string)
		kind := zipkinmodel.Undetermined
		if ok {
			// Check if it is one of the kinds that are defined by conventions.
			switch tracetranslator.OpenTracingSpanKind(spanKindStr) {
			case tracetranslator.OpenTracingSpanKindClient:
				kind = zipkinmodel.Client
			case tracetranslator.OpenTracingSpanKindServer:
				kind = zipkinmodel.Server
			case tracetranslator.OpenTracingSpanKindConsumer:
				kind = zipkinmodel.Consumer
			case tracetranslator.OpenTracingSpanKindProducer:
				kind = zipkinmodel.Producer
			default:
				// Unknown kind. Keep the attribute, but return Undetermined to our caller.
				return zipkinmodel.Undetermined
			}
			// The special attribute is no longer needed, delete it.
			delete(s.Attributes, tracetranslator.TagSpanKind)
		}
		return kind
	}
	return zipkinmodel.Undetermined
}

type zipkinDirection bool

const (
	isLocalEndpoint  zipkinDirection = true
	isRemoteEndpoint zipkinDirection = false
)

func serviceNameOrDefault(node *commonpb.Node, defaultServiceName string) string {
	if node == nil || node.ServiceInfo == nil {
		return defaultServiceName
	}

	if node.ServiceInfo.Name == "" {
		return defaultServiceName
	}

	return node.ServiceInfo.Name
}

func OCSpanDataToZipkin(
	node *commonpb.Node,
	s *trace.SpanData,
	defaultServiceName string,
) (zc zipkinmodel.SpanModel) {

	// Per call to zipkinEndpointFromAttributes at most 3 attribute keys can be
	// made redundant, give a hint when calling make that we expect at most 6
	// items on the map.
	redundantKeys := make(map[string]bool, 6)
	localEndpointServiceName := serviceNameOrDefault(node, defaultServiceName)
	localEndpoint := zipkinEndpointFromAttributes(
		s.Attributes, localEndpointServiceName, isLocalEndpoint, redundantKeys)

	remoteServiceName := ""
	if remoteServiceEntry, ok := s.Attributes[RemoteEndpointServiceName]; ok {
		if remoteServiceName, ok = remoteServiceEntry.(string); ok {
			redundantKeys[RemoteEndpointServiceName] = true
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
