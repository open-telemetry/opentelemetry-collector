// Copyright 2020, OpenTelemetry Authors
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
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var sampled = true

func InternalTracesToZipkinSpans(td pdata.Traces) ([]*zipkinmodel.SpanModel, error) {
	resourceSpans := td.ResourceSpans()

	if resourceSpans.Len() == 0 {
		return nil, nil
	}

	zSpans := make([]*zipkinmodel.SpanModel, 0, td.SpanCount())

	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		if rs.IsNil() {
			continue
		}

		batch, err := resourceSpansToZipkinSpans(rs, td.SpanCount()/resourceSpans.Len())
		if err != nil {
			return nil, err
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

	if resource.IsNil() && ilss.Len() == 0 {
		return nil, nil
	}

	localServiceName := resourceToZipkinEndpointServiceName(resource)
	zSpans := make([]*zipkinmodel.SpanModel, 0, estSpanCount)
	for i := 0; i < ilss.Len(); i++ {
		ils := ilss.At(i)
		if ils.IsNil() {
			continue
		}
		spans := ils.Spans()
		for j := 0; j < spans.Len(); j++ {
			span, err := spanToZipkinSpan(spans.At(j), localServiceName)
			if err == nil {
				zSpans = append(zSpans, span)
			}
		}
	}

	return zSpans, nil
}

func spanToZipkinSpan(span pdata.Span, localServiceName string) (*zipkinmodel.SpanModel, error) {
	zs := zipkinmodel.SpanModel{}
	hi, lo, err := tracetranslator.BytesToUInt64TraceID(span.TraceID().Bytes())
	if err != nil {
		return nil, err
	}
	zs.TraceID = zipkinmodel.TraceID{
		High: hi,
		Low:  lo,
	}
	idVal, err := tracetranslator.BytesToUInt64SpanID(span.SpanID().Bytes())
	if err != nil {
		return nil, err
	}
	zs.ID = zipkinmodel.ID(idVal)
	if len(span.ParentSpanID().Bytes()) > 0 {
		idVal, err := tracetranslator.BytesToUInt64SpanID(span.ParentSpanID().Bytes())
		if err != nil {
			return nil, err
		}
		id := zipkinmodel.ID(idVal)
		zs.ParentID = &id
	}
	zs.Sampled = &sampled
	zs.Name = span.Name()
	zs.Timestamp = internal.UnixNanoToTime(span.StartTime())
	if span.EndTime() != 0 {
		zs.Duration = time.Duration(span.EndTime() - span.StartTime())
	}
	zs.Kind = spanKindToZipkinKind(span.Kind())
	return &zs, nil
}

func resourceToZipkinEndpointServiceName(resource pdata.Resource) string {
	var serviceName string
	if resource.IsNil() {
		serviceName = tracetranslator.ResourceNotSet
	}
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		serviceName = tracetranslator.ResourceNoAttrs
	}
	if sn, ok := attrs.Get(conventions.AttributeServiceName); ok {
		serviceName = sn.StringVal()
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
