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

package zipkin

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/pkg/errors"
)

var (
	// ZipkinV1 friendly convertion errors
	msgZipkinV1JSONUnmarshalError = "zipkinv1"
	msgZipkinV1TraceIDError       = "zipkinV1 span traceId"
	msgZipkinV1SpanIDError        = "zipkinV1 span id"
	msgZipkinV1ParentIDError      = "zipkinV1 span parentId"
	// Generic hex to ID convertion errors
	errHexTraceIDWrongLen = errors.New("hex traceId span has wrong length (expected 16 or 32)")
	errHexTraceIDParsing  = errors.New("failed to parse hex traceId")
	errHexTraceIDZero     = errors.New("traceId is zero")
	errHexIDWrongLen      = errors.New("hex Id has wrong length (expected 16)")
	errHexIDParsing       = errors.New("failed to parse hex Id")
	errHexIDZero          = errors.New("Id is zero")
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

// V1JSONBatchToOCProto converts a JSON blob with a list of Zipkin v1 spans to OC Proto.
func V1JSONBatchToOCProto(blob []byte) ([]*agenttracepb.ExportTraceServiceRequest, error) {
	var zSpans []*zipkinV1Span
	if err := json.Unmarshal(blob, &zSpans); err != nil {
		return nil, errors.WithMessage(err, msgZipkinV1JSONUnmarshalError)
	}

	ocSpansAndParsedAnnotations := make([]ocSpanAndParsedAnnotations, 0, len(zSpans))
	for _, zSpan := range zSpans {
		ocSpan, parsedAnnotations, err := zipkinV1ToOCSpan(zSpan)
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

func zipkinToOCProtoBatch(ocSpansAndParsedAnnotations []ocSpanAndParsedAnnotations) ([]*agenttracepb.ExportTraceServiceRequest, error) {
	// Service to batch maps the service name to the trace request with the corresponding node.
	svcToBatch := make(map[string]*agenttracepb.ExportTraceServiceRequest)
	for _, curr := range ocSpansAndParsedAnnotations {
		req := getOrCreateNodeRequest(svcToBatch, curr.parsedAnnotations.Endpoint)
		req.Spans = append(req.Spans, curr.ocSpan)
	}

	batches := make([]*agenttracepb.ExportTraceServiceRequest, 0, len(svcToBatch))
	for _, v := range svcToBatch {
		batches = append(batches, v)
	}
	return batches, nil
}

func zipkinV1ToOCSpan(zSpan *zipkinV1Span) (*tracepb.Span, *annotationParseResult, error) {
	traceID, err := hexTraceIDToOCTraceID(zSpan.TraceID)
	if err != nil {
		return nil, nil, errors.WithMessage(err, msgZipkinV1TraceIDError)
	}
	spanID, err := hexIDToOCID(zSpan.ID)
	if err != nil {
		return nil, nil, errors.WithMessage(err, msgZipkinV1SpanIDError)
	}
	var parentID []byte
	if zSpan.ParentID != "" {
		id, err := hexIDToOCID(zSpan.ParentID)
		if err != nil {
			return nil, nil, errors.WithMessage(err, msgZipkinV1ParentIDError)
		}
		parentID = id
	}

	parsedAnnotations := parseZipkinV1Annotations(zSpan.Annotations)
	attributes, localComponent := zipkinV1BinAnnotationsToOCAttributes(zSpan.BinaryAnnotations)
	if parsedAnnotations.Endpoint.ServiceName == unknownServiceName && localComponent != "" {
		parsedAnnotations.Endpoint.ServiceName = localComponent
	}
	var startTime, endTime *timestamp.Timestamp
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
		Kind:         parsedAnnotations.Kind,
		TimeEvents:   parsedAnnotations.TimeEvents,
		StartTime:    startTime,
		EndTime:      endTime,
		Attributes:   attributes,
	}

	if zSpan.Name != "" {
		ocSpan.Name = &tracepb.TruncatableString{Value: zSpan.Name}
	}

	return ocSpan, parsedAnnotations, nil
}

func zipkinV1BinAnnotationsToOCAttributes(binAnnotations []*binaryAnnotation) (attributes *tracepb.Span_Attributes, fallbackServiceName string) {
	if len(binAnnotations) == 0 {
		return nil, ""
	}

	var localComponent string
	attributeMap := make(map[string]*tracepb.AttributeValue)
	for _, binAnnotation := range binAnnotations {
		if binAnnotation.Endpoint != nil && binAnnotation.Endpoint.ServiceName != "" {
			fallbackServiceName = binAnnotation.Endpoint.ServiceName
		}
		pbAttrib := &tracepb.AttributeValue{}
		if iValue, err := strconv.ParseInt(binAnnotation.Value, 10, 64); err == nil {
			pbAttrib.Value = &tracepb.AttributeValue_IntValue{IntValue: iValue}
		} else if bValue, err := strconv.ParseBool(binAnnotation.Value); err == nil {
			pbAttrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: bValue}
		} else {
			// For now all else go to string
			pbAttrib.Value = &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: binAnnotation.Value}}
		}

		key := binAnnotation.Key
		if key == zipkincore.LOCAL_COMPONENT {
			// TODO: (@pjanotti) add reference to OpenTracing and change related tags to use them
			key = "component"
			localComponent = binAnnotation.Value
		}
		attributeMap[key] = pbAttrib
	}

	if len(attributeMap) == 0 {
		return nil, ""
	}

	if fallbackServiceName == "" {
		fallbackServiceName = localComponent
	}

	attributes = &tracepb.Span_Attributes{
		AttributeMap: attributeMap,
	}

	return attributes, fallbackServiceName
}

// annotationParseResult stores the results of examining the original annotations,
// this way multiple passes on the annotations are not needed.
type annotationParseResult struct {
	Endpoint            *endpoint
	TimeEvents          *tracepb.Span_TimeEvents
	Kind                tracepb.Span_SpanKind
	EarlyAnnotationTime *timestamp.Timestamp
	LateAnnotationTime  *timestamp.Timestamp
}

// Unknown service name works both as a default value and a flag to indicate that a valid endpoint was found.
const unknownServiceName = "unknown-service"

func parseZipkinV1Annotations(annotations []*annotation) *annotationParseResult {
	// Zipkin V1 annotations have a timestamp so they fit well with OC TimeEvent
	earlyAnnotationTimestamp := int64(math.MaxInt64)
	lateAnnotationTimestamp := int64(math.MinInt64)
	res := &annotationParseResult{}
	timeEvents := make([]*tracepb.Span_TimeEvent, 0, len(annotations))
	for _, currAnnotation := range annotations {
		if currAnnotation == nil && currAnnotation.Value == "" {
			continue
		}

		endpointName := unknownServiceName
		if currAnnotation.Endpoint != nil && currAnnotation.Endpoint.ServiceName != "" {
			endpointName = currAnnotation.Endpoint.ServiceName
		}

		// Specially important annotations used by zipkin v1 these are the most important ones.
		switch currAnnotation.Value {
		case "cs":
			fallthrough
		case "cr":
			if res.Kind == tracepb.Span_SPAN_KIND_UNSPECIFIED {
				res.Kind = tracepb.Span_CLIENT
			}
			fallthrough
		case "ss":
			fallthrough
		case "sr":
			if res.Kind == tracepb.Span_SPAN_KIND_UNSPECIFIED {
				res.Kind = tracepb.Span_SERVER
			}
			if res.Endpoint == nil && endpointName != unknownServiceName {
				res.Endpoint = currAnnotation.Endpoint
			}
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

		timeEvent := &tracepb.Span_TimeEvent{
			Time: ts,
			// More economically we could use a tracepb.Span_TimeEvent_Message, however, it will mean the loss of some information.
			// Using the more expensive annotation until/if something cheaper is needed.
			Value: &tracepb.Span_TimeEvent_Annotation_{
				Annotation: &tracepb.Span_TimeEvent_Annotation{
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							currAnnotation.Value: {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{
										Value: endpointName,
									},
								},
							},
						},
					},
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

	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return traceID, nil
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

	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, idValue)
	return id, nil
}

func epochMicrosecondsToTimestamp(msecs int64) *timestamp.Timestamp {
	if msecs <= 0 {
		return nil
	}
	t := &timestamp.Timestamp{}
	t.Seconds = msecs / 1e6
	t.Nanos = int32(msecs%1e6) * 1e3
	return t
}

func getOrCreateNodeRequest(m map[string]*agenttracepb.ExportTraceServiceRequest, endpoint *endpoint) *agenttracepb.ExportTraceServiceRequest {
	// this private function assumes that the caller never passes an nil endpoint
	nodeKey := endpoint.string()
	req := m[nodeKey]
	if req != nil {
		return req
	}

	req = &agenttracepb.ExportTraceServiceRequest{
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
