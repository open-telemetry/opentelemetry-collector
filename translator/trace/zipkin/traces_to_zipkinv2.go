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
	"net"
	"strconv"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var sampled = true

// InternalTracesToZipkinSpans translates internal trace data into Zipkin v2 spans.
// Returns a slice of Zipkin SpanModel's.
func InternalTracesToZipkinSpans(td pdata.Traces) ([]*zipkinmodel.SpanModel, error) {

	resourceSpans := td.ResourceSpans()
	if resourceSpans.Len() == 0 {
		return nil, nil
	}

	zSpans := make([]*zipkinmodel.SpanModel, 0, td.SpanCount())

	for i := 0; i < resourceSpans.Len(); i++ {
		batch, err := resourceSpansToZipkinSpans(resourceSpans.At(i), td.SpanCount()/resourceSpans.Len())
		if err != nil {
			return zSpans, err
		}
		if batch != nil {
			zSpans = append(zSpans, batch...)
		}
	}

	return zSpans, nil
}

func resourceSpansToZipkinSpans(rs pdata.ResourceSpans, estSpanCount int) ([]*zipkinmodel.SpanModel, error) {
	resource := rs.Resource()
	ilss := rs.InstrumentationLibrarySpans()

	if resource.Attributes().Len() == 0 && ilss.Len() == 0 {
		return nil, nil
	}

	localServiceName, zTags := resourceToZipkinEndpointServiceNameAndAttributeMap(resource)

	zSpans := make([]*zipkinmodel.SpanModel, 0, estSpanCount)
	for i := 0; i < ilss.Len(); i++ {
		ils := ilss.At(i)
		extractInstrumentationLibraryTags(ils.InstrumentationLibrary(), zTags)
		spans := ils.Spans()
		for j := 0; j < spans.Len(); j++ {
			zSpan, err := spanToZipkinSpan(spans.At(j), localServiceName, zTags)
			if err != nil {
				return zSpans, err
			}
			zSpans = append(zSpans, zSpan)
		}
	}

	return zSpans, nil
}

func extractInstrumentationLibraryTags(il pdata.InstrumentationLibrary, zTags map[string]string) {
	if ilName := il.Name(); ilName != "" {
		zTags[tracetranslator.TagInstrumentationName] = ilName
	}
	if ilVer := il.Version(); ilVer != "" {
		zTags[tracetranslator.TagInstrumentationVersion] = ilVer
	}
}

func spanToZipkinSpan(
	span pdata.Span,
	localServiceName string,
	zTags map[string]string,
) (*zipkinmodel.SpanModel, error) {

	tags := aggregateSpanTags(span, zTags)

	zs := &zipkinmodel.SpanModel{}

	if !span.TraceID().IsValid() {
		return zs, errors.New("TraceID is invalid")
	}
	zs.TraceID = convertTraceID(span.TraceID())
	if !span.SpanID().IsValid() {
		return zs, errors.New("SpanID is invalid")
	}
	zs.ID = convertSpanID(span.SpanID())

	if len(span.TraceState()) > 0 {
		tags[tracetranslator.TagW3CTraceState] = string(span.TraceState())
	}

	if span.ParentSpanID().IsValid() {
		id := convertSpanID(span.ParentSpanID())
		zs.ParentID = &id
	}

	zs.Sampled = &sampled
	zs.Name = span.Name()
	zs.Timestamp = pdata.UnixNanoToTime(span.StartTime())
	if span.EndTime() != 0 {
		zs.Duration = time.Duration(span.EndTime() - span.StartTime())
	}
	zs.Kind = spanKindToZipkinKind(span.Kind())
	if span.Kind() == pdata.SpanKindINTERNAL {
		tags[tracetranslator.TagSpanKind] = "internal"
	}

	redundantKeys := make(map[string]bool, 8)
	zs.LocalEndpoint = zipkinEndpointFromTags(tags, localServiceName, false, redundantKeys)
	zs.RemoteEndpoint = zipkinEndpointFromTags(tags, "", true, redundantKeys)

	removeRedundentTags(redundantKeys, tags)

	status := span.Status()
	tags[tracetranslator.TagStatusCode] = status.Code().String()
	if status.Message() != "" {
		tags[tracetranslator.TagStatusMsg] = status.Message()
		if int32(status.Code()) > 0 {
			zs.Err = fmt.Errorf("%s", status.Message())
		}
	}

	if err := spanEventsToZipkinAnnotations(span.Events(), zs); err != nil {
		return nil, err
	}
	if err := spanLinksToZipkinTags(span.Links(), tags); err != nil {
		return nil, err
	}

	zs.Tags = tags

	return zs, nil
}

func aggregateSpanTags(span pdata.Span, zTags map[string]string) map[string]string {
	tags := make(map[string]string)
	for key, val := range zTags {
		tags[key] = val
	}
	spanTags := attributeMapToStringMap(span.Attributes())
	for key, val := range spanTags {
		tags[key] = val
	}
	return tags
}

func spanEventsToZipkinAnnotations(events pdata.SpanEventSlice, zs *zipkinmodel.SpanModel) error {
	if events.Len() > 0 {
		zAnnos := make([]zipkinmodel.Annotation, events.Len())
		for i := 0; i < events.Len(); i++ {
			event := events.At(i)
			if event.Attributes().Len() == 0 && event.DroppedAttributesCount() == 0 {
				zAnnos[i] = zipkinmodel.Annotation{
					Timestamp: pdata.UnixNanoToTime(event.Timestamp()),
					Value:     event.Name(),
				}
			} else {
				jsonStr, err := json.Marshal(tracetranslator.AttributeMapToMap(event.Attributes()))
				if err != nil {
					return err
				}
				zAnnos[i] = zipkinmodel.Annotation{
					Timestamp: pdata.UnixNanoToTime(event.Timestamp()),
					Value: fmt.Sprintf(tracetranslator.SpanEventDataFormat, event.Name(), jsonStr,
						event.DroppedAttributesCount()),
				}
			}
		}
		zs.Annotations = zAnnos
	}
	return nil
}

func spanLinksToZipkinTags(links pdata.SpanLinkSlice, zTags map[string]string) error {
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		key := fmt.Sprintf("otlp.link.%d", i)
		jsonStr, err := json.Marshal(tracetranslator.AttributeMapToMap(link.Attributes()))
		if err != nil {
			return err
		}
		zTags[key] = fmt.Sprintf(tracetranslator.SpanLinkDataFormat, link.TraceID().HexString(),
			link.SpanID().HexString(), link.TraceState(), jsonStr, link.DroppedAttributesCount())
	}
	return nil
}

func attributeMapToStringMap(attrMap pdata.AttributeMap) map[string]string {
	rawMap := make(map[string]string)
	attrMap.ForEach(func(k string, v pdata.AttributeValue) {
		rawMap[k] = tracetranslator.AttributeValueToString(v, false)
	})
	return rawMap
}

func removeRedundentTags(redundantKeys map[string]bool, zTags map[string]string) {
	for k, v := range redundantKeys {
		if v {
			delete(zTags, k)
		}
	}
}

func resourceToZipkinEndpointServiceNameAndAttributeMap(
	resource pdata.Resource,
) (serviceName string, zTags map[string]string) {
	zTags = make(map[string]string)
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		return tracetranslator.ResourceNoServiceName, zTags
	}

	attrs.ForEach(func(k string, v pdata.AttributeValue) {
		zTags[k] = tracetranslator.AttributeValueToString(v, false)
	})

	serviceName = extractZipkinServiceName(zTags)
	return serviceName, zTags
}

func extractZipkinServiceName(zTags map[string]string) string {
	var serviceName string
	if sn, ok := zTags[conventions.AttributeServiceName]; ok {
		serviceName = sn
		delete(zTags, conventions.AttributeServiceName)
	} else if fn, ok := zTags[conventions.AttributeFaasName]; ok {
		serviceName = fn
		delete(zTags, conventions.AttributeFaasName)
		zTags[tracetranslator.TagServiceNameSource] = conventions.AttributeFaasName
	} else if fn, ok := zTags[conventions.AttributeK8sDeployment]; ok {
		serviceName = fn
		delete(zTags, conventions.AttributeK8sDeployment)
		zTags[tracetranslator.TagServiceNameSource] = conventions.AttributeK8sDeployment
	} else if fn, ok := zTags[conventions.AttributeProcessExecutableName]; ok {
		serviceName = fn
		delete(zTags, conventions.AttributeProcessExecutableName)
		zTags[tracetranslator.TagServiceNameSource] = conventions.AttributeProcessExecutableName
	} else {
		serviceName = tracetranslator.ResourceNoServiceName
	}
	return serviceName
}

func spanKindToZipkinKind(kind pdata.SpanKind) zipkinmodel.Kind {
	switch kind {
	case pdata.SpanKindCLIENT:
		return zipkinmodel.Client
	case pdata.SpanKindSERVER:
		return zipkinmodel.Server
	case pdata.SpanKindPRODUCER:
		return zipkinmodel.Producer
	case pdata.SpanKindCONSUMER:
		return zipkinmodel.Consumer
	default:
		return zipkinmodel.Undetermined
	}
}

func zipkinEndpointFromTags(
	zTags map[string]string,
	localServiceName string,
	remoteEndpoint bool,
	redundantKeys map[string]bool,
) (endpoint *zipkinmodel.Endpoint) {

	serviceName := localServiceName
	if peerSvc, ok := zTags[conventions.AttributePeerService]; ok && remoteEndpoint {
		serviceName = peerSvc
		redundantKeys[conventions.AttributePeerService] = true
	}

	var ipKey, portKey string
	if remoteEndpoint {
		ipKey, portKey = conventions.AttributeNetPeerIP, conventions.AttributeNetPeerPort
	} else {
		ipKey, portKey = conventions.AttributeNetHostIP, conventions.AttributeNetHostPort
	}

	var ip net.IP
	ipv6Selected := false
	if ipStr, ok := zTags[ipKey]; ok {
		ipv6Selected = isIPv6Address(ipStr)
		ip = net.ParseIP(ipStr)
		redundantKeys[ipKey] = true
	}

	var port uint64
	if portStr, ok := zTags[portKey]; ok {
		port, _ = strconv.ParseUint(portStr, 10, 16)
		redundantKeys[portKey] = true
	}

	if serviceName == "" && ip == nil {
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

func isIPv6Address(ipStr string) bool {
	for i := 0; i < len(ipStr); i++ {
		if ipStr[i] == ':' {
			return true
		}
	}
	return false
}

func convertTraceID(t pdata.TraceID) zipkinmodel.TraceID {
	h, l := tracetranslator.TraceIDToUInt64Pair(t)
	return zipkinmodel.TraceID{High: h, Low: l}
}

func convertSpanID(s pdata.SpanID) zipkinmodel.ID {
	return zipkinmodel.ID(tracetranslator.BytesToUInt64SpanID(s.Bytes()))
}
