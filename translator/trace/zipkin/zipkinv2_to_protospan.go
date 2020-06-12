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

package zipkin

import (
	"net"
	"strconv"
	"strings"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/internal"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func V2BatchToOCProto(zipkinSpans []*zipkinmodel.SpanModel) (reqs []consumerdata.TraceData, err error) {
	// *commonpb.Node instances have unique addresses hence
	// for grouping within a map, we'll use the .String() value
	byNodeGrouping := make(map[string][]*tracepb.Span)
	uniqueNodes := make([]*commonpb.Node, 0, len(zipkinSpans))
	// Now translate them into tracepb.Span
	for _, zspan := range zipkinSpans {
		if zspan == nil {
			continue
		}
		span, node := zipkinSpanToTraceSpan(zspan)
		key := node.String()
		if _, alreadyAdded := byNodeGrouping[key]; !alreadyAdded {
			uniqueNodes = append(uniqueNodes, node)
		}
		byNodeGrouping[key] = append(byNodeGrouping[key], span)
	}

	for _, node := range uniqueNodes {
		key := node.String()
		spans := byNodeGrouping[key]
		if len(spans) == 0 {
			// Should never happen but nonetheless be cautious
			// not to send blank spans.
			continue
		}
		reqs = append(reqs, consumerdata.TraceData{
			Node:  node,
			Spans: spans,
		})
		delete(byNodeGrouping, key)
	}

	return reqs, nil
}

func zipkinSpanToTraceSpan(zs *zipkinmodel.SpanModel) (*tracepb.Span, *commonpb.Node) {
	traceID := tracetranslator.UInt64ToByteTraceID(zs.TraceID.High, zs.TraceID.Low)
	var parentSpanID []byte
	if zs.ParentID != nil {
		parentSpanID = tracetranslator.UInt64ToByteSpanID(uint64(*zs.ParentID))
	}

	attributes, ocStatus := zipkinTagsToTraceAttributes(zs.Tags, zs.Kind)

	pbs := &tracepb.Span{
		TraceId:      traceID,
		SpanId:       tracetranslator.UInt64ToByteSpanID(uint64(zs.ID)),
		ParentSpanId: parentSpanID,
		Name:         &tracepb.TruncatableString{Value: zs.Name},
		StartTime:    internal.TimeToTimestamp(zs.Timestamp),
		EndTime:      internal.TimeToTimestamp(zs.Timestamp.Add(zs.Duration)),
		Kind:         zipkinSpanKindToProtoSpanKind(zs.Kind),
		Status:       ocStatus,
		Attributes:   attributes,
		TimeEvents:   zipkinAnnotationsToProtoTimeEvents(zs.Annotations),
	}

	node := nodeFromZipkinEndpoints(zs, pbs)
	setTimestampsIfUnset(pbs)

	return pbs, node
}

func nodeFromZipkinEndpoints(zs *zipkinmodel.SpanModel, pbs *tracepb.Span) *commonpb.Node {
	if zs.LocalEndpoint == nil && zs.RemoteEndpoint == nil {
		return nil
	}

	node := new(commonpb.Node)
	var endpointMap map[string]string

	// Retrieve and make use of the local endpoint
	if lep := zs.LocalEndpoint; lep != nil {
		node.ServiceInfo = &commonpb.ServiceInfo{
			Name: lep.ServiceName,
		}
		endpointMap = zipkinEndpointIntoAttributes(lep, endpointMap, isLocalEndpoint)
	}

	// Retrieve and make use of the remote endpoint
	if rep := zs.RemoteEndpoint; rep != nil {
		endpointMap = zipkinEndpointIntoAttributes(rep, endpointMap, isRemoteEndpoint)
	}

	if endpointMap != nil {
		if pbs.Attributes == nil {
			pbs.Attributes = &tracepb.Span_Attributes{}
		}
		if pbs.Attributes.AttributeMap == nil {
			pbs.Attributes.AttributeMap = make(
				map[string]*tracepb.AttributeValue, len(endpointMap))
		}

		// Delete the redundant serviceName key since it is already on the node.
		delete(endpointMap, LocalEndpointServiceName)
		attrbMap := pbs.Attributes.AttributeMap
		for key, value := range endpointMap {
			attrbMap[key] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: value},
				},
			}
		}
	}

	return node
}

var blankIP net.IP

// zipkinEndpointIntoAttributes extracts information from s zipkin endpoint struct
// and puts it into a map with pre-defined keys.
func zipkinEndpointIntoAttributes(
	ep *zipkinmodel.Endpoint,
	into map[string]string,
	endpointType zipkinDirection,
) map[string]string {

	if into == nil {
		into = make(map[string]string)
	}

	var ipv4Key, ipv6Key, portKey, serviceNameKey string
	if endpointType == isLocalEndpoint {
		ipv4Key, ipv6Key = LocalEndpointIPv4, LocalEndpointIPv6
		portKey, serviceNameKey = LocalEndpointPort, LocalEndpointServiceName
	} else {
		ipv4Key, ipv6Key = RemoteEndpointIPv4, RemoteEndpointIPv6
		portKey, serviceNameKey = RemoteEndpointPort, RemoteEndpointServiceName
	}
	if ep.IPv4 != nil && !ep.IPv4.Equal(blankIP) {
		into[ipv4Key] = ep.IPv4.String()
	}
	if ep.IPv6 != nil && !ep.IPv6.Equal(blankIP) {
		into[ipv6Key] = ep.IPv6.String()
	}
	if ep.Port > 0 {
		into[portKey] = strconv.Itoa(int(ep.Port))
	}
	if serviceName := ep.ServiceName; serviceName != "" {
		into[serviceNameKey] = serviceName
	}
	return into
}

func zipkinSpanKindToProtoSpanKind(skind zipkinmodel.Kind) tracepb.Span_SpanKind {
	switch strings.ToUpper(string(skind)) {
	case "CLIENT":
		return tracepb.Span_CLIENT
	case "SERVER":
		return tracepb.Span_SERVER
	default:
		return tracepb.Span_SPAN_KIND_UNSPECIFIED
	}
}

func zipkinAnnotationsToProtoTimeEvents(zas []zipkinmodel.Annotation) *tracepb.Span_TimeEvents {
	if len(zas) == 0 {
		return nil
	}
	tevs := make([]*tracepb.Span_TimeEvent, 0, len(zas))
	for _, za := range zas {
		if tev := zipkinAnnotationToProtoAnnotation(za); tev != nil {
			tevs = append(tevs, tev)
		}
	}
	if len(tevs) == 0 {
		return nil
	}
	return &tracepb.Span_TimeEvents{
		TimeEvent: tevs,
	}
}

var blankAnnotation zipkinmodel.Annotation

func zipkinAnnotationToProtoAnnotation(zas zipkinmodel.Annotation) *tracepb.Span_TimeEvent {
	if zas == blankAnnotation {
		return nil
	}
	return &tracepb.Span_TimeEvent{
		Time: internal.TimeToTimestamp(zas.Timestamp),
		Value: &tracepb.Span_TimeEvent_Annotation_{
			Annotation: &tracepb.Span_TimeEvent_Annotation{
				Description: &tracepb.TruncatableString{Value: zas.Value},
			},
		},
	}
}

func zipkinTagsToTraceAttributes(tags map[string]string, skind zipkinmodel.Kind) (*tracepb.Span_Attributes, *tracepb.Status) {
	// Produce and Consumer span kinds are not representable in OpenCensus format.
	// We will represent them using TagSpanKind attribute, according to OpenTracing
	// conventions. Check if it is one of those span kinds.
	var spanKindTagVal tracetranslator.OpenTracingSpanKind
	switch skind {
	case zipkinmodel.Producer:
		spanKindTagVal = tracetranslator.OpenTracingSpanKindProducer
	case zipkinmodel.Consumer:
		spanKindTagVal = tracetranslator.OpenTracingSpanKindConsumer
	}

	if len(tags) == 0 && spanKindTagVal == "" {
		// No input tags and no need to add a span kind tag. Keep attributes map empty.
		return nil, nil
	}

	sMapper := &statusMapper{}
	amap := make(map[string]*tracepb.AttributeValue, len(tags))
	for key, value := range tags {

		pbAttrib := parseAnnotationValue(value)

		if drop := sMapper.fromAttribute(key, pbAttrib); drop {
			continue
		}

		amap[key] = pbAttrib
	}

	status := sMapper.ocStatus()

	if spanKindTagVal != "" {
		// Set the previously translated span kind attribute (see top of this function).
		// We do this after the "tags" map is translated so that we will overwrite
		// the attribute if it exists.
		amap[tracetranslator.TagSpanKind] = &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: string(spanKindTagVal)},
			},
		}
	}

	return &tracepb.Span_Attributes{AttributeMap: amap}, status
}
