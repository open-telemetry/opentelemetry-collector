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
	"sort"
	"strings"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

// Custome Sort on
type byOTLPTypes []*zipkinmodel.SpanModel

func (b byOTLPTypes) Len() int {
	return len(b)
}

func (b byOTLPTypes) Less(i, j int) bool {
	diff := strings.Compare(extractLocalServiceName(b[i]), extractLocalServiceName(b[j]))
	if diff != 0 {
		return diff <= 0
	}
	diff = strings.Compare(extractInstrumentationLibrary(b[i]), extractInstrumentationLibrary(b[j]))
	return diff <= 0
}

func (b byOTLPTypes) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// V2BatchToTraces translates Zipkin v2 spans into internal trace data.
func V2BatchToTraces(zipkinSpans []*zipkinmodel.SpanModel) (pdata.Traces, error) {
	traceData := pdata.NewTraces()
	if len(zipkinSpans) == 0 {
		return traceData, nil
	}

	sort.Sort(byOTLPTypes(zipkinSpans))

	rss := traceData.ResourceSpans()
	prevServiceName := ""
	rsCount := rss.Len()
	var curRscSpans pdata.ResourceSpans
	for _, zspan := range zipkinSpans {
		if zspan == nil {
			continue
		}
		localServiceName := extractLocalServiceName(zspan)
		if localServiceName != prevServiceName {
			prevServiceName = localServiceName
			rss.Resize(rsCount + 1)
			curRscSpans = rss.At(rsCount)
			curRscSpans.InitEmpty()
			rsCount++
			populateResourceFromZipkinSpan(zspan, localServiceName, curRscSpans.Resource())
		}
	}

	return traceData, nil
}

func populateResourceFromZipkinSpan(zspan *zipkinmodel.SpanModel, localServiceName string, resource pdata.Resource) {
	if tracetranslator.ResourceNotSet == localServiceName {
		return
	}

	resource.InitEmpty()
	if tracetranslator.ResourceNoAttrs == localServiceName {
		return
	}

	tags := zspan.Tags
	if len(tags) == 0 {
		resource.Attributes().InsertString(conventions.AttributeServiceName, localServiceName)
		return
	}

	snSource := tags[tracetranslator.TagServiceNameSource]
	if snSource == "" {
		resource.Attributes().InsertString(conventions.AttributeServiceName, localServiceName)
	} else {
		resource.Attributes().InsertString(snSource, localServiceName)
	}

	for _, key := range conventions.GetResourceSemanticConventionAttributeNames() {
		value, ok := tags[key]
		if ok {
			resource.Attributes().InsertString(key, value)
		}
	}
}

func extractLocalServiceName(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || zspan.LocalEndpoint == nil {
		return tracetranslator.ResourceNotSet
	}
	return zspan.LocalEndpoint.ServiceName
}

func extractInstrumentationLibrary(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || len(zspan.Tags) == 0 {
		return ""
	}
	return zspan.Tags[tracetranslator.TagInstrumentationName]
}
