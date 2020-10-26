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
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

// v1ThriftBatchToOCProto converts Zipkin v1 spans to OC Proto.
func v1ThriftBatchToOCProto(zSpans []*zipkincore.Span) ([]consumerdata.TraceData, error) {
	ocSpansAndParsedAnnotations := make([]ocSpanAndParsedAnnotations, 0, len(zSpans))
	for _, zSpan := range zSpans {
		ocSpan, parsedAnnotations := zipkinV1ThriftToOCSpan(zSpan)
		ocSpansAndParsedAnnotations = append(ocSpansAndParsedAnnotations, ocSpanAndParsedAnnotations{
			ocSpan:            ocSpan,
			parsedAnnotations: parsedAnnotations,
		})
	}

	return zipkinToOCProtoBatch(ocSpansAndParsedAnnotations)
}

func zipkinV1ThriftToOCSpan(zSpan *zipkincore.Span) (*tracepb.Span, *annotationParseResult) {
	traceIDHigh := int64(0)
	if zSpan.TraceIDHigh != nil {
		traceIDHigh = *zSpan.TraceIDHigh
	}

	// TODO: (@pjanotti) ideally we should error here instead of generating invalid OC proto
	// however per https://go.opentelemetry.io/collector/issues/349
	// failures on the receivers in general are silent at this moment, so letting them
	// proceed for now. We should validate the traceID, spanID and parentID are good with
	// OC proto requirements.
	traceID := tracetranslator.Int64ToByteTraceID(traceIDHigh, zSpan.TraceID)
	spanID := tracetranslator.Int64ToByteSpanID(zSpan.ID)
	var parentID []byte
	if zSpan.ParentID != nil {
		parentIDBytes := tracetranslator.Int64ToByteSpanID(*zSpan.ParentID)
		parentID = parentIDBytes[:]
	}

	parsedAnnotations := parseZipkinV1ThriftAnnotations(zSpan.Annotations)
	attributes, ocStatus, localComponent := zipkinV1ThriftBinAnnotationsToOCAttributes(zSpan.BinaryAnnotations)
	if parsedAnnotations.Endpoint.ServiceName == unknownServiceName && localComponent != "" {
		parsedAnnotations.Endpoint.ServiceName = localComponent
	}

	var startTime, endTime *timestamppb.Timestamp
	if zSpan.Timestamp == nil {
		startTime = parsedAnnotations.EarlyAnnotationTime
		endTime = parsedAnnotations.LateAnnotationTime
	} else {
		startTime = epochMicrosecondsToTimestamp(*zSpan.Timestamp)
		var duration int64
		if zSpan.Duration != nil {
			duration = *zSpan.Duration
		}
		endTime = epochMicrosecondsToTimestamp(*zSpan.Timestamp + duration)
	}

	ocSpan := &tracepb.Span{
		TraceId:      traceID[:],
		SpanId:       spanID[:],
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

	return ocSpan, parsedAnnotations
}

func parseZipkinV1ThriftAnnotations(ztAnnotations []*zipkincore.Annotation) *annotationParseResult {
	annotations := make([]*annotation, 0, len(ztAnnotations))
	for _, ztAnnot := range ztAnnotations {
		annot := &annotation{
			Timestamp: ztAnnot.Timestamp,
			Value:     ztAnnot.Value,
			Endpoint:  toTranslatorEndpoint(ztAnnot.Host),
		}
		annotations = append(annotations, annot)
	}
	return parseZipkinV1Annotations(annotations)
}

func toTranslatorEndpoint(e *zipkincore.Endpoint) *endpoint {
	if e == nil {
		return nil
	}

	var ipv4, ipv6 string
	if e.Ipv4 != 0 {
		ipv4 = net.IPv4(byte(e.Ipv4>>24), byte(e.Ipv4>>16), byte(e.Ipv4>>8), byte(e.Ipv4)).String()
	}
	if len(e.Ipv6) != 0 {
		ipv6 = net.IP(e.Ipv6).String()
	}
	return &endpoint{
		ServiceName: e.ServiceName,
		IPv4:        ipv4,
		IPv6:        ipv6,
		Port:        int32(e.Port),
	}
}

var trueByteSlice = []byte{1}

func zipkinV1ThriftBinAnnotationsToOCAttributes(ztBinAnnotations []*zipkincore.BinaryAnnotation) (attributes *tracepb.Span_Attributes, status *tracepb.Status, fallbackServiceName string) {
	if len(ztBinAnnotations) == 0 {
		return nil, nil, ""
	}

	sMapper := &statusMapper{}
	var localComponent string
	attributeMap := make(map[string]*tracepb.AttributeValue)
	for _, binaryAnnotation := range ztBinAnnotations {
		pbAttrib := &tracepb.AttributeValue{}
		binAnnotationType := binaryAnnotation.AnnotationType
		if binaryAnnotation.Host != nil {
			fallbackServiceName = binaryAnnotation.Host.ServiceName
		}
		switch binaryAnnotation.AnnotationType {
		case zipkincore.AnnotationType_BOOL:
			isTrue := bytes.Equal(binaryAnnotation.Value, trueByteSlice)
			pbAttrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: isTrue}
		case zipkincore.AnnotationType_BYTES:
			bytesStr := base64.StdEncoding.EncodeToString(binaryAnnotation.Value)
			pbAttrib.Value = &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: bytesStr}}
		case zipkincore.AnnotationType_DOUBLE:
			if d, err := bytesFloat64ToFloat64(binaryAnnotation.Value); err != nil {
				pbAttrib.Value = strAttributeForError(err)
			} else {
				pbAttrib.Value = &tracepb.AttributeValue_DoubleValue{DoubleValue: d}
			}
		case zipkincore.AnnotationType_I16:
			if i, err := bytesInt16ToInt64(binaryAnnotation.Value); err != nil {
				pbAttrib.Value = strAttributeForError(err)
			} else {
				pbAttrib.Value = &tracepb.AttributeValue_IntValue{IntValue: i}
			}
		case zipkincore.AnnotationType_I32:
			if i, err := bytesInt32ToInt64(binaryAnnotation.Value); err != nil {
				pbAttrib.Value = strAttributeForError(err)
			} else {
				pbAttrib.Value = &tracepb.AttributeValue_IntValue{IntValue: i}
			}
		case zipkincore.AnnotationType_I64:
			if i, err := bytesInt64ToInt64(binaryAnnotation.Value); err != nil {
				pbAttrib.Value = strAttributeForError(err)
			} else {
				pbAttrib.Value = &tracepb.AttributeValue_IntValue{IntValue: i}
			}
		case zipkincore.AnnotationType_STRING:
			pbAttrib.Value = &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: string(binaryAnnotation.Value)}}
		default:
			err := fmt.Errorf("unknown zipkin v1 binary annotation type (%d)", int(binAnnotationType))
			pbAttrib.Value = strAttributeForError(err)
		}

		key := binaryAnnotation.Key
		if key == zipkincore.LOCAL_COMPONENT {
			// TODO: (@pjanotti) add reference to OpenTracing and change related tags to use them
			key = "component"
			localComponent = string(binaryAnnotation.Value)
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

var errNotEnoughBytes = errors.New("not enough bytes representing the number")

func bytesInt16ToInt64(b []byte) (int64, error) {
	const minSliceLength = 2
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint16(b[:minSliceLength])), nil
}

func bytesInt32ToInt64(b []byte) (int64, error) {
	const minSliceLength = 4
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint32(b[:minSliceLength])), nil
}

func bytesInt64ToInt64(b []byte) (int64, error) {
	const minSliceLength = 8
	if len(b) < minSliceLength {
		return 0, errNotEnoughBytes
	}
	return int64(binary.BigEndian.Uint64(b[:minSliceLength])), nil
}

func bytesFloat64ToFloat64(b []byte) (float64, error) {
	const minSliceLength = 8
	if len(b) < minSliceLength {
		return 0.0, errNotEnoughBytes
	}
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits), nil
}

func strAttributeForError(err error) *tracepb.AttributeValue_StringValue {
	return &tracepb.AttributeValue_StringValue{
		StringValue: &tracepb.TruncatableString{
			Value: "<" + err.Error() + ">",
		},
	}
}
