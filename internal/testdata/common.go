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
	resourceAttributes1 = map[string]pdata.AttributeValue{"resource-attr": pdata.NewAttributeValueString("resource-attr-val-1")}
	resourceAttributes2 = map[string]pdata.AttributeValue{"resource-attr": pdata.NewAttributeValueString("resource-attr-val-2")}
	spanEventAttributes = map[string]pdata.AttributeValue{"span-event-attr": pdata.NewAttributeValueString("span-event-attr-val")}
	spanLinkAttributes  = map[string]pdata.AttributeValue{"span-link-attr": pdata.NewAttributeValueString("span-link-attr-val")}
	spanAttributes      = map[string]pdata.AttributeValue{"span-attr": pdata.NewAttributeValueString("span-attr-val")}
)

const (
	TestLabelKey1       = "label-1"
	TestLabelValue1     = "label-value-1"
	TestLabelKey2       = "label-2"
	TestLabelValue2     = "label-value-2"
	TestLabelKey3       = "label-3"
	TestLabelValue3     = "label-value-3"
	TestAttachmentKey   = "exemplar-attachment"
	TestAttachmentValue = "exemplar-attachment-value"
)

func initResourceAttributes1(dest pdata.AttributeMap) {
	dest.InitFromMap(resourceAttributes1)
}

func initResourceAttributes2(dest pdata.AttributeMap) {
	dest.InitFromMap(resourceAttributes2)
}

func initSpanAttributes(dest pdata.AttributeMap) {
	dest.InitFromMap(spanAttributes)
}

func initSpanEventAttributes(dest pdata.AttributeMap) {
	dest.InitFromMap(spanEventAttributes)
}

func initSpanLinkAttributes(dest pdata.AttributeMap) {
	dest.InitFromMap(spanLinkAttributes)
}

func initMetricLabels1(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{TestLabelKey1: TestLabelValue1})
}

func initMetricLabels12(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{TestLabelKey1: TestLabelValue1, TestLabelKey2: TestLabelValue2}).Sort()
}

func initMetricLabels13(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{TestLabelKey1: TestLabelValue1, TestLabelKey3: TestLabelValue3}).Sort()
}

func initMetricLabels2(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{TestLabelKey2: TestLabelValue2})
}

func initMetricAttachment(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{TestAttachmentKey: TestAttachmentValue})
}
