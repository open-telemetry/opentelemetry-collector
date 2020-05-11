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
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/pkg/errors"

	"github.com/open-telemetry/opentelemetry-collector/internal"
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
	errNilSpan = errors.New("expected a non-nil span")
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
	attrMap map[string]*tracepb.AttributeValue,
	serviceName string,
	endpointType zipkinDirection,
	redundantKeys map[string]bool,
) (endpoint *zipkinmodel.Endpoint) {

	if attrMap == nil {
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

	if ipv4Str, ok := extractStringAttribute(attrMap, ipv4Key); ok {
		ip = net.ParseIP(ipv4Str)
		redundantKeys[ipv4Key] = true
	} else if ipv6Str, ok := extractStringAttribute(attrMap, ipv6Key); ok {
		ip = net.ParseIP(ipv6Str)
		ipv6Selected = true
		redundantKeys[ipv6Key] = true
	}

	var port uint64
	if portStr, ok := extractStringAttribute(attrMap, portKey); ok {
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

func extractStringAttribute(attributes map[string]*tracepb.AttributeValue, key string) (string, bool) {
	av, ok := attributes[key]
	if !ok || av == nil || av.Value == nil {
		return "", false
	}

	if value, ok := av.Value.(*tracepb.AttributeValue_StringValue); ok {
		return value.StringValue.GetValue(), true
	}

	return "", false
}

func convertTraceID(t []byte) zipkinmodel.TraceID {
	h, l, _ := tracetranslator.BytesToUInt64TraceID(t)
	return zipkinmodel.TraceID{High: h, Low: l}
}

func convertSpanID(s []byte) zipkinmodel.ID {
	id, _ := tracetranslator.BytesToUInt64SpanID(s)
	return zipkinmodel.ID(id)
}

// spanKind returns the Kind and a boolean indicating if the kind was extracted from the attributes.
func spanKind(s *tracepb.Span) (zipkinmodel.Kind, bool) {
	// First try span kinds that are set directly in SpanKind field.
	switch s.Kind {
	case tracepb.Span_CLIENT:
		return zipkinmodel.Client, false
	case tracepb.Span_SERVER:
		return zipkinmodel.Server, false
	}

	// SpanKind==SpanKindUnspecified, check if TagSpanKind attribute is set.
	// This can happen if span kind had no equivalent in OC, so we could represent it in
	// the SpanKind. In that case we had set a special attribute TagSpanKind in
	// the span to preserve the span kind. Now we will do a reverse translation.
	if s.Attributes == nil || s.Attributes.AttributeMap == nil {
		return zipkinmodel.Undetermined, false
	}

	spanKindStr, ok := extractStringAttribute(s.Attributes.AttributeMap, tracetranslator.TagSpanKind)
	if ok {
		// Check if it is one of the kinds that are defined by conventions.
		switch tracetranslator.OpenTracingSpanKind(spanKindStr) {
		case tracetranslator.OpenTracingSpanKindClient:
			return zipkinmodel.Client, true
		case tracetranslator.OpenTracingSpanKindServer:
			return zipkinmodel.Server, true
		case tracetranslator.OpenTracingSpanKindConsumer:
			return zipkinmodel.Consumer, true
		case tracetranslator.OpenTracingSpanKindProducer:
			return zipkinmodel.Producer, true
		default:
			// Unknown kind. Keep the attribute, but return Undetermined to our caller.
			return zipkinmodel.Undetermined, false
		}
	}
	return zipkinmodel.Undetermined, false
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

func OCSpanProtoToZipkin(
	node *commonpb.Node,
	resource *resourcepb.Resource,
	s *tracepb.Span,
	defaultServiceName string,
) (*zipkinmodel.SpanModel, error) {
	if s == nil {
		return nil, errNilSpan
	}

	var attrMap map[string]*tracepb.AttributeValue
	if s.Attributes != nil {
		attrMap = s.Attributes.AttributeMap
	}

	// Per call to zipkinEndpointFromAttributes at most 3 attribute keys can be made redundant,
	// give a hint when calling make that we expect at most 2*3+2 items on the map.
	redundantKeys := make(map[string]bool, 8)
	localEndpointServiceName := serviceNameOrDefault(node, defaultServiceName)
	localEndpoint := zipkinEndpointFromAttributes(attrMap, localEndpointServiceName, isLocalEndpoint, redundantKeys)

	remoteServiceName, okServiceName := extractStringAttribute(attrMap, RemoteEndpointServiceName)
	if okServiceName {
		redundantKeys[RemoteEndpointServiceName] = true
	}
	remoteEndpoint := zipkinEndpointFromAttributes(attrMap, remoteServiceName, isRemoteEndpoint, redundantKeys)

	sk, spanKindFromAttributes := spanKind(s)
	if spanKindFromAttributes {
		redundantKeys[tracetranslator.TagSpanKind] = true
	}
	startTime := internal.TimestampToTime(s.StartTime)
	z := &zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID: convertTraceID(s.TraceId),
			ID:      convertSpanID(s.SpanId),
			Sampled: &sampledTrue,
		},
		Kind:           sk,
		Name:           s.Name.GetValue(),
		Timestamp:      startTime,
		Shared:         false,
		LocalEndpoint:  localEndpoint,
		RemoteEndpoint: remoteEndpoint,
	}

	resourceLabelesCount := 0
	if resource != nil {
		resourceLabelesCount = len(resource.Labels)
	}
	//
	z.Tags = make(map[string]string, len(attrMap)+2+resourceLabelesCount)
	resourceToSpanAttributes(resource, z.Tags)

	if s.ParentSpanId != nil {
		id := convertSpanID(s.ParentSpanId)
		z.ParentID = &id
	}

	if endTime := internal.TimestampToTime(s.EndTime); !startTime.IsZero() && !endTime.IsZero() {
		z.Duration = endTime.Sub(startTime)
	}

	// construct Tags from s.Attributes and s.Status.
	attributesToSpanTags(attrMap, redundantKeys, z.Tags)
	status := s.Status
	if status != nil && (status.Code != 0 || status.Message != "") {
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
	z.Annotations = timeEventsToSpanAnnotations(s.TimeEvents)

	return z, nil
}

func resourceToSpanAttributes(resource *resourcepb.Resource, tags map[string]string) {
	if resource != nil {
		for key, value := range resource.Labels {
			tags[key] = value
		}
	}
}

func timeEventsToSpanAnnotations(timeEvents *tracepb.Span_TimeEvents) []zipkinmodel.Annotation {
	var annotations []zipkinmodel.Annotation
	// construct Annotations from s.Annotations and s.MessageEvents.
	if timeEvents != nil && len(timeEvents.TimeEvent) != 0 {
		annotations = make([]zipkinmodel.Annotation, 0, len(timeEvents.TimeEvent))
		for _, te := range timeEvents.TimeEvent {
			if te == nil || te.Value == nil {
				continue
			}

			switch tme := te.Value.(type) {
			case *tracepb.Span_TimeEvent_Annotation_:
				annotations = append(annotations, zipkinmodel.Annotation{
					Timestamp: internal.TimestampToTime(te.Time),
					Value:     tme.Annotation.Description.GetValue(),
				})
			case *tracepb.Span_TimeEvent_MessageEvent_:
				a := zipkinmodel.Annotation{
					Timestamp: internal.TimestampToTime(te.Time),
					Value:     tme.MessageEvent.Type.String(),
				}
				annotations = append(annotations, a)

			}
		}
	}
	return annotations
}

func attributesToSpanTags(attrMap map[string]*tracepb.AttributeValue, redundantKeys map[string]bool, tags map[string]string) {
	if len(attrMap) == 0 {
		return
	}
	for key, value := range attrMap {
		if redundantKeys[key] {
			// Already represented by something other than an attribute,
			// skip it.
			continue
		}

		switch v := value.Value.(type) {
		case *tracepb.AttributeValue_BoolValue:
			if v.BoolValue {
				tags[key] = "true"
			} else {
				tags[key] = "false"
			}
		case *tracepb.AttributeValue_IntValue:
			tags[key] = strconv.FormatInt(v.IntValue, 10)
		case *tracepb.AttributeValue_DoubleValue:
			tags[key] = strconv.FormatFloat(v.DoubleValue, 'f', -1, 64)
		case *tracepb.AttributeValue_StringValue:
			tags[key] = v.StringValue.GetValue()
		}
	}
}
