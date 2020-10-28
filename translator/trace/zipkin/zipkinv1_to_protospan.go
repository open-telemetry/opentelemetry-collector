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

package zipkin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var (
	// ZipkinV1 friendly conversion errors
	msgZipkinV1JSONUnmarshalError = "zipkinv1"
	msgZipkinV1TraceIDError       = "zipkinV1 span traceId"
	msgZipkinV1SpanIDError        = "zipkinV1 span id"
	msgZipkinV1ParentIDError      = "zipkinV1 span parentId"
	// Generic hex to ID conversion errors
	errHexTraceIDWrongLen = errors.New("hex traceId span has wrong length (expected 16 or 32)")
	errHexTraceIDParsing  = errors.New("failed to parse hex traceId")
	errHexTraceIDZero     = errors.New("traceId is zero")
	errHexIDWrongLen      = errors.New("hex Id has wrong length (expected 16)")
	errHexIDParsing       = errors.New("failed to parse hex Id")
	errHexIDZero          = errors.New("ID is zero")
)

// Trace translation from Zipkin V1 is a bit of special case since there is no model
// defined in golang for Zipkin V1 spans and there is no need to define one here, given
// that the zipkinV1Span defined below is as defined at:
// https://zipkin.io/zipkin-api/zipkin-api.yaml
type zipkinV1Span struct {
	TraceID           string              `json:"traceId"`
	Name              string              `json:"name,omitempty"`
	ParentID          string              `json:"parentId,omitempty"`
	ID                string              `json:"id"`
	Timestamp         int64               `json:"timestamp"`
	Duration          int64               `json:"duration"`
	Debug             bool                `json:"debug,omitempty"`
	Annotations       []*annotation       `json:"annotations,omitempty"`
	BinaryAnnotations []*binaryAnnotation `json:"binaryAnnotations,omitempty"`
}

// endpoint structure used by zipkinV1Span.
type endpoint struct {
	ServiceName string `json:"serviceName"`
	IPv4        string `json:"ipv4"`
	IPv6        string `json:"ipv6"`
	Port        int32  `json:"port"`
}

// annotation struct used by zipkinV1Span.
type annotation struct {
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Endpoint  *endpoint `json:"endpoint"`
}

// binaryAnnotation used by zipkinV1Span.
type binaryAnnotation struct {
	Key      string    `json:"key"`
	Value    string    `json:"value"`
	Endpoint *endpoint `json:"endpoint"`
}

// v1JSONBatchToOCProto converts a JSON blob with a list of Zipkin v1 spans to OC Proto.
func v1JSONBatchToOCProto(blob []byte, parseStringTags bool) ([]consumerdata.TraceData, error) {
	var zSpans []*zipkinV1Span
	if err := json.Unmarshal(blob, &zSpans); err != nil {
		return nil, fmt.Errorf("%s: %w", msgZipkinV1JSONUnmarshalError, err)
	}

	ocSpansAndParsedAnnotations := make([]ocSpanAndParsedAnnotations, 0, len(zSpans))
	for _, zSpan := range zSpans {
		ocSpan, parsedAnnotations, err := zipkinV1ToOCSpan(zSpan, parseStringTags)
		if err != nil {
			// error from internal package function, it already wraps the error to give better context.
			return nil, err
		}
		ocSpansAndParsedAnnotations = append(ocSpansAndParsedAnnotations, ocSpanAndParsedAnnotations{
			ocSpan:            ocSpan,
			parsedAnnotations: parsedAnnotations,
		})
	}

	return zipkinToOCProtoBatch(ocSpansAndParsedAnnotations)
}

type ocSpanAndParsedAnnotations struct {
	ocSpan            *tracepb.Span
	parsedAnnotations *annotationParseResult
}

func zipkinToOCProtoBatch(ocSpansAndParsedAnnotations []ocSpanAndParsedAnnotations) ([]consumerdata.TraceData, error) {
	// Service to batch maps the service name to the trace request with the corresponding node.
	svcToTD := make(map[string]*consumerdata.TraceData)
	for _, curr := range ocSpansAndParsedAnnotations {
		req := getOrCreateNodeRequest(svcToTD, curr.parsedAnnotations.Endpoint)
		req.Spans = append(req.Spans, curr.ocSpan)
	}

	tds := make([]consumerdata.TraceData, 0, len(svcToTD))
	for _, v := range svcToTD {
		tds = append(tds, *v)
	}
	return tds, nil
}

func zipkinV1ToOCSpan(zSpan *zipkinV1Span, parseStringTags bool) (*tracepb.Span, *annotationParseResult, error) {
	traceID, err := hexTraceIDToOCTraceID(zSpan.TraceID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", msgZipkinV1TraceIDError, err)
	}
	spanID, err := hexIDToOCID(zSpan.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", msgZipkinV1SpanIDError, err)
	}
	var parentID []byte
	if zSpan.ParentID != "" {
		id, err := hexIDToOCID(zSpan.ParentID)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", msgZipkinV1ParentIDError, err)
		}
		parentID = id
	}

	parsedAnnotations := parseZipkinV1Annotations(zSpan.Annotations)
	attributes, ocStatus, localComponent := zipkinV1BinAnnotationsToOCAttributes(zSpan.BinaryAnnotations, parseStringTags)
	if parsedAnnotations.Endpoint.ServiceName == unknownServiceName && localComponent != "" {
		parsedAnnotations.Endpoint.ServiceName = localComponent
	}
	var startTime, endTime *timestamppb.Timestamp
	if zSpan.Timestamp == 0 {
		startTime = parsedAnnotations.EarlyAnnotationTime
		endTime = parsedAnnotations.LateAnnotationTime
	} else {
		startTime = epochMicrosecondsToTimestamp(zSpan.Timestamp)
		endTime = epochMicrosecondsToTimestamp(zSpan.Timestamp + zSpan.Duration)
	}

	ocSpan := &tracepb.Span{
		TraceId:      traceID,
		SpanId:       spanID,
		ParentSpanId: parentID,
		Status:       ocStatus,
		Kind:         parsedAnnotations.Kind,
		TimeEvents:   parsedAnnotations.TimeEvents,
		StartTime:    startTime,
		EndTime:      endTime,
		Attributes:   attributes,
	}

	if zSpan.Name != "" {
		ocSpan.Name = &tracepb.TruncatableString{Value: zSpan.Name}
	}

	setSpanKind(ocSpan, parsedAnnotations.Kind, parsedAnnotations.ExtendedKind)
	setTimestampsIfUnset(ocSpan)

	return ocSpan, parsedAnnotations, nil
}

func setSpanKind(ocSpan *tracepb.Span, kind tracepb.Span_SpanKind, extendedKind tracetranslator.OpenTracingSpanKind) {
	if kind == tracepb.Span_SPAN_KIND_UNSPECIFIED &&
		extendedKind != tracetranslator.OpenTracingSpanKindUnspecified {
		// Span kind has no equivalent in OC, so we cannot represent it in the Kind field.
		// We will set a TagSpanKind attribute in the span. This will successfully transfer
		// in the pipeline until it reaches the exporter which is responsible for
		// reverse translation.
		if ocSpan.Attributes == nil {
			ocSpan.Attributes = &tracepb.Span_Attributes{}
		}
		if ocSpan.Attributes.AttributeMap == nil {
			ocSpan.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, 1)
		}
		ocSpan.Attributes.AttributeMap[tracetranslator.TagSpanKind] =
			&tracepb.AttributeValue{Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: string(extendedKind)},
			}}
	}
}

func zipkinV1BinAnnotationsToOCAttributes(binAnnotations []*binaryAnnotation, parseStringTags bool) (attributes *tracepb.Span_Attributes, status *tracepb.Status, fallbackServiceName string) {
	if len(binAnnotations) == 0 {
		return nil, nil, ""
	}

	sMapper := &statusMapper{}
	var localComponent string
	attributeMap := make(map[string]*tracepb.AttributeValue)
	for _, binAnnotation := range binAnnotations {

		if binAnnotation.Endpoint != nil && binAnnotation.Endpoint.ServiceName != "" {
			fallbackServiceName = binAnnotation.Endpoint.ServiceName
		}

		pbAttrib := parseAnnotationValue(binAnnotation.Value, parseStringTags)

		key := binAnnotation.Key

		if key == zipkincore.LOCAL_COMPONENT {
			// TODO: (@pjanotti) add reference to OpenTracing and change related tags to use them
			key = "component"
			localComponent = binAnnotation.Value
		}

		if drop := sMapper.fromAttribute(key, pbAttrib); drop {
			continue
		}

		attributeMap[key] = pbAttrib
	}

	status = sMapper.ocStatus()

	if len(attributeMap) == 0 {
		return nil, status, ""
	}

	if fallbackServiceName == "" {
		fallbackServiceName = localComponent
	}

	attributes = &tracepb.Span_Attributes{
		AttributeMap: attributeMap,
	}

	return attributes, status, fallbackServiceName
}

func parseAnnotationValue(value string, parseStringTags bool) *tracepb.AttributeValue {
	pbAttrib := &tracepb.AttributeValue{}

	if parseStringTags {
		switch tracetranslator.DetermineValueType(value, false) {
		case pdata.AttributeValueINT:
			iValue, _ := strconv.ParseInt(value, 10, 64)
			pbAttrib.Value = &tracepb.AttributeValue_IntValue{IntValue: iValue}
		case pdata.AttributeValueDOUBLE:
			fValue, _ := strconv.ParseFloat(value, 64)
			pbAttrib.Value = &tracepb.AttributeValue_DoubleValue{DoubleValue: fValue}
		case pdata.AttributeValueBOOL:
			bValue, _ := strconv.ParseBool(value)
			pbAttrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: bValue}
		default:
			pbAttrib.Value = &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: value}}
		}
	} else {
		pbAttrib.Value = &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: value}}
	}

	return pbAttrib
}

// annotationParseResult stores the results of examining the original annotations,
// this way multiple passes on the annotations are not needed.
type annotationParseResult struct {
	Endpoint            *endpoint
	TimeEvents          *tracepb.Span_TimeEvents
	Kind                tracepb.Span_SpanKind
	ExtendedKind        tracetranslator.OpenTracingSpanKind
	EarlyAnnotationTime *timestamppb.Timestamp
	LateAnnotationTime  *timestamppb.Timestamp
}

// Unknown service name works both as a default value and a flag to indicate that a valid endpoint was found.
const unknownServiceName = "unknown-service"

func parseZipkinV1Annotations(annotations []*annotation) *annotationParseResult {
	// Zipkin V1 annotations have a timestamp so they fit well with OC TimeEvent
	earlyAnnotationTimestamp := int64(math.MaxInt64)
	lateAnnotationTimestamp := int64(math.MinInt64)
	res := &annotationParseResult{}
	timeEvents := make([]*tracepb.Span_TimeEvent, 0, len(annotations))

	// We want to set the span kind from the first annotation that contains information
	// about the span kind. This flags ensures we only set span kind once from
	// the first annotation.
	spanKindIsSet := false

	for _, currAnnotation := range annotations {
		if currAnnotation == nil || currAnnotation.Value == "" {
			continue
		}

		endpointName := unknownServiceName
		if currAnnotation.Endpoint != nil && currAnnotation.Endpoint.ServiceName != "" {
			endpointName = currAnnotation.Endpoint.ServiceName
		}

		// Check if annotation has span kind information.
		annotationHasSpanKind := false
		switch currAnnotation.Value {
		case "cs", "cr", "ms", "mr", "ss", "sr":
			annotationHasSpanKind = true
		}

		// Populate the endpoint if it is not already populated and current endpoint
		// has a service name and span kind.
		if res.Endpoint == nil && endpointName != unknownServiceName && annotationHasSpanKind {
			res.Endpoint = currAnnotation.Endpoint
		}

		if !spanKindIsSet && annotationHasSpanKind {
			// We have not yet populated span kind, do it now.
			// Translate from Zipkin span kind stored in Value field to Kind/ExternalKind
			// pair of internal fields.
			switch currAnnotation.Value {
			case "cs", "cr":
				res.Kind = tracepb.Span_CLIENT
				res.ExtendedKind = tracetranslator.OpenTracingSpanKindClient

			case "ms":
				// "ms" and "mr" are PRODUCER and CONSUMER kinds which have no equivalent
				// representation in OC. We keep res.Kind unspecified and will use
				// ExtendedKind for translations.
				res.ExtendedKind = tracetranslator.OpenTracingSpanKindProducer

			case "mr":
				res.ExtendedKind = tracetranslator.OpenTracingSpanKindConsumer

			case "ss", "sr":
				res.Kind = tracepb.Span_SERVER
				res.ExtendedKind = tracetranslator.OpenTracingSpanKindServer
			}

			// Remember that we populated the span kind, so that we don't do it again.
			spanKindIsSet = true
		}

		ts := epochMicrosecondsToTimestamp(currAnnotation.Timestamp)
		if currAnnotation.Timestamp < earlyAnnotationTimestamp {
			earlyAnnotationTimestamp = currAnnotation.Timestamp
			res.EarlyAnnotationTime = ts
		}
		if currAnnotation.Timestamp > lateAnnotationTimestamp {
			lateAnnotationTimestamp = currAnnotation.Timestamp
			res.LateAnnotationTime = ts
		}

		if annotationHasSpanKind {
			// If this annotation is for the send/receive timestamps, no need to create the annotation
			continue
		}

		timeEvent := &tracepb.Span_TimeEvent{
			Time: ts,
			// More economically we could use a tracepb.Span_TimeEvent_Message, however, it will mean the loss of some information.
			// Using the more expensive annotation until/if something cheaper is needed.
			Value: &tracepb.Span_TimeEvent_Annotation_{
				Annotation: &tracepb.Span_TimeEvent_Annotation{
					Description: &tracepb.TruncatableString{Value: currAnnotation.Value},
				},
			},
		}

		timeEvents = append(timeEvents, timeEvent)
	}

	if len(timeEvents) > 0 {
		res.TimeEvents = &tracepb.Span_TimeEvents{TimeEvent: timeEvents}
	}

	if res.Endpoint == nil {
		res.Endpoint = &endpoint{
			ServiceName: unknownServiceName,
		}
	}

	return res
}

func hexTraceIDToOCTraceID(hex string) ([]byte, error) {
	// Per info at https://zipkin.io/zipkin-api/zipkin-api.yaml it should be 16 or 32 characters
	hexLen := len(hex)
	if hexLen != 16 && hexLen != 32 {
		return nil, errHexTraceIDWrongLen
	}

	var high, low uint64
	var err error
	if hexLen == 32 {
		if high, err = strconv.ParseUint(hex[:16], 16, 64); err != nil {
			return nil, errHexTraceIDParsing
		}
	}

	if low, err = strconv.ParseUint(hex[hexLen-16:], 16, 64); err != nil {
		return nil, errHexTraceIDParsing
	}

	if high == 0 && low == 0 {
		return nil, errHexTraceIDZero
	}

	tidBytes := tracetranslator.UInt64ToByteTraceID(high, low)
	return tidBytes[:], nil
}

func hexIDToOCID(hex string) ([]byte, error) {
	// Per info at https://zipkin.io/zipkin-api/zipkin-api.yaml it should be 16 characters
	if len(hex) != 16 {
		return nil, errHexIDWrongLen
	}

	idValue, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return nil, errHexIDParsing
	}

	if idValue == 0 {
		return nil, errHexIDZero
	}

	idBytes := tracetranslator.UInt64ToByteSpanID(idValue)
	return idBytes[:], nil
}

func epochMicrosecondsToTimestamp(msecs int64) *timestamppb.Timestamp {
	if msecs <= 0 {
		return nil
	}
	t := &timestamppb.Timestamp{}
	t.Seconds = msecs / 1e6
	t.Nanos = int32(msecs%1e6) * 1e3
	return t
}

func getOrCreateNodeRequest(m map[string]*consumerdata.TraceData, endpoint *endpoint) *consumerdata.TraceData {
	// this private function assumes that the caller never passes an nil endpoint
	nodeKey := endpoint.string()
	req := m[nodeKey]

	if req != nil {
		return req
	}

	req = &consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: endpoint.ServiceName},
		},
	}

	if attributeMap := endpoint.createAttributeMap(); attributeMap != nil {
		req.Node.Attributes = attributeMap
	}

	m[nodeKey] = req

	return req
}

func (ep *endpoint) string() string {
	return fmt.Sprintf("%s-%s-%s-%d", ep.ServiceName, ep.IPv4, ep.IPv6, ep.Port)
}

func (ep *endpoint) createAttributeMap() map[string]string {
	if ep.IPv4 == "" && ep.IPv6 == "" && ep.Port == 0 {
		return nil
	}

	attributeMap := make(map[string]string, 3)
	if ep.IPv4 != "" {
		attributeMap["ipv4"] = ep.IPv4
	}
	if ep.IPv6 != "" {
		attributeMap["ipv6"] = ep.IPv6
	}
	if ep.Port != 0 {
		attributeMap["port"] = strconv.Itoa(int(ep.Port))
	}
	return attributeMap
}

func setTimestampsIfUnset(span *tracepb.Span) {
	// zipkin allows timestamp to be unset, but opentelemetry-collector expects it to have a value.
	// If this is unset, the conversion from open census to the internal trace format breaks
	// what should be an identity transformation oc -> internal -> oc
	if span.StartTime == nil {
		now := timestamppb.New(time.Now())
		span.StartTime = now
		span.EndTime = now

		if span.Attributes == nil {
			span.Attributes = &tracepb.Span_Attributes{}
		}
		if span.Attributes.AttributeMap == nil {
			span.Attributes.AttributeMap = make(map[string]*tracepb.AttributeValue, 1)
		}
		span.Attributes.AttributeMap[StartTimeAbsent] = &tracepb.AttributeValue{
			Value: &tracepb.AttributeValue_BoolValue{
				BoolValue: true,
			}}
	}
}
