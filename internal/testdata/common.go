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
	resourceAttributes1 = pdata.NewMapFromRaw(map[string]interface{}{"resource-attr": "resource-attr-val-1"})
	resourceAttributes2 = pdata.NewMapFromRaw(map[string]interface{}{"resource-attr": "resource-attr-val-2"})
	spanEventAttributes = pdata.NewMapFromRaw(map[string]interface{}{"span-event-attr": "span-event-attr-val"})
	spanLinkAttributes  = pdata.NewMapFromRaw(map[string]interface{}{"span-link-attr": "span-link-attr-val"})
	spanAttributes      = pdata.NewMapFromRaw(map[string]interface{}{"span-attr": "span-attr-val"})
	metricAttachment    = pdata.NewMapFromRaw(map[string]interface{}{"exemplar-attachment": "exemplar-attachment-value"})
)

const (
	TestLabelKey1   = "label-1"
	TestLabelValue1 = "label-value-1"
	TestLabelKey2   = "label-2"
	TestLabelValue2 = "label-value-2"
	TestLabelKey3   = "label-3"
	TestLabelValue3 = "label-value-3"
)

func initResourceAttributes1(dest pdata.Map) {
	dest.Clear()
	resourceAttributes1.CopyTo(dest)
}

func initResourceAttributes2(dest pdata.Map) {
	dest.Clear()
	resourceAttributes2.CopyTo(dest)
}

func initSpanAttributes(dest pdata.Map) {
	dest.Clear()
	spanAttributes.CopyTo(dest)
}

func initSpanEventAttributes(dest pdata.Map) {
	dest.Clear()
	spanEventAttributes.CopyTo(dest)
}

func initSpanLinkAttributes(dest pdata.Map) {
	dest.Clear()
	spanLinkAttributes.CopyTo(dest)
}

func initMetricAttachment(dest pdata.Map) {
	dest.Clear()
	metricAttachment.CopyTo(dest)
}

func initMetricAttributes1(dest pdata.Map) {
	dest.Clear()
	dest.InsertString(TestLabelKey1, TestLabelValue1)
}

func initMetricAttributes12(dest pdata.Map) {
	initMetricAttributes1(dest)
	dest.InsertString(TestLabelKey2, TestLabelValue2)
}

func initMetricAttributes13(dest pdata.Map) {
	initMetricAttributes1(dest)
	dest.InsertString(TestLabelKey3, TestLabelValue3)
}

func initMetricAttributes2(dest pdata.Map) {
	dest.Clear()
	dest.InsertString(TestLabelKey2, TestLabelValue2)
}
