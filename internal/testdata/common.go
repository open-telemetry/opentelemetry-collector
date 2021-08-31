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

package testdata

import (
	"go.opentelemetry.io/collector/model/pdata"
)

var (
	resourceAttributes1 = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"resource-attr": pdata.NewAttributeValueString("resource-attr-val-1")})
	resourceAttributes2 = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"resource-attr": pdata.NewAttributeValueString("resource-attr-val-2")})
	spanEventAttributes = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"span-event-attr": pdata.NewAttributeValueString("span-event-attr-val")})
	spanLinkAttributes  = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"span-link-attr": pdata.NewAttributeValueString("span-link-attr-val")})
	spanAttributes      = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"span-attr": pdata.NewAttributeValueString("span-attr-val")})
	metricAttachment    = pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{"exemplar-attachment": pdata.NewAttributeValueString("exemplar-attachment-value")})
)

const (
	TestLabelKey1   = "label-1"
	TestLabelValue1 = "label-value-1"
	TestLabelKey2   = "label-2"
	TestLabelValue2 = "label-value-2"
	TestLabelKey3   = "label-3"
	TestLabelValue3 = "label-value-3"
)

func initResourceAttributes1(dest pdata.AttributeMap) {
	dest.Clear()
	resourceAttributes1.CopyTo(dest)
}

func initResourceAttributes2(dest pdata.AttributeMap) {
	dest.Clear()
	resourceAttributes2.CopyTo(dest)
}

func initSpanAttributes(dest pdata.AttributeMap) {
	dest.Clear()
	spanAttributes.CopyTo(dest)
}

func initSpanEventAttributes(dest pdata.AttributeMap) {
	dest.Clear()
	spanEventAttributes.CopyTo(dest)
}

func initSpanLinkAttributes(dest pdata.AttributeMap) {
	dest.Clear()
	spanLinkAttributes.CopyTo(dest)
}

func initMetricAttachment(dest pdata.AttributeMap) {
	dest.Clear()
	metricAttachment.CopyTo(dest)
}

func initMetricAttributes1(dest pdata.AttributeMap) {
	dest.Clear()
	dest.InsertString(TestLabelKey1, TestLabelValue1)
}

func initMetricAttributes12(dest pdata.AttributeMap) {
	initMetricAttributes1(dest)
	dest.InsertString(TestLabelKey2, TestLabelValue2)
}

func initMetricAttributes13(dest pdata.AttributeMap) {
	initMetricAttributes1(dest)
	dest.InsertString(TestLabelKey3, TestLabelValue3)
}

func initMetricAttributes2(dest pdata.AttributeMap) {
	dest.Clear()
	dest.InsertString(TestLabelKey2, TestLabelValue2)
}
