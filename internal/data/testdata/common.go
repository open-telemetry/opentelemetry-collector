// Copyright 2020 OpenTelemetry Authors
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

package testdata

import (
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

var (
	resourceAttributes1 = map[string]data.AttributeValue{"resource-attr": data.NewAttributeValueString("resource-attr-val-1")}
	resourceAttributes2 = map[string]data.AttributeValue{"resource-attr": data.NewAttributeValueString("resource-attr-val-2")}
	spanEventAttributes = map[string]data.AttributeValue{"span-event-attr": data.NewAttributeValueString("span-event-attr-val")}
	spanLinkAttributes  = map[string]data.AttributeValue{"span-link-attr": data.NewAttributeValueString("span-link-attr-val")}
	spanAttributes      = map[string]data.AttributeValue{"span-attr": data.NewAttributeValueString("span-attr-val")}
)

func initResourceAttributes1(dest data.AttributeMap) {
	dest.InitFromMap(resourceAttributes1)
}

func generateOtlpResourceAttributes1() []*otlpcommon.AttributeKeyValue {
	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         "resource-attr",
			StringValue: "resource-attr-val-1",
		},
	}
}

func initResourceAttributes2(dest data.AttributeMap) {
	dest.InitFromMap(resourceAttributes2)
}

func generateOtlpResourceAttributes2() []*otlpcommon.AttributeKeyValue {
	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         "resource-attr",
			StringValue: "resource-attr-val-2",
		},
	}
}

func initSpanAttributes(dest data.AttributeMap) {
	dest.InitFromMap(spanAttributes)
}

func generateOtlpSpanAttributes() []*otlpcommon.AttributeKeyValue {
	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         "span-attr",
			StringValue: "span-attr-val",
		},
	}
}

func initSpanEventAttributes(dest data.AttributeMap) {
	dest.InitFromMap(spanEventAttributes)
}

func generateOtlpSpanEventAttributes() []*otlpcommon.AttributeKeyValue {
	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         "span-event-attr",
			StringValue: "span-event-attr-val",
		},
	}
}

func initSpanLinkAttributes(dest data.AttributeMap) {
	dest.InitFromMap(spanLinkAttributes)
}

func generateOtlpSpanLinkAttributes() []*otlpcommon.AttributeKeyValue {
	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         "span-link-attr",
			StringValue: "span-link-attr-val",
		},
	}
}
