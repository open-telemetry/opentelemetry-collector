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
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	testLabelKey2   = "label-2"
	testLabelValue2 = "label-value-2"
)

func initResourceAttributes1(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString("resource-attr", "resource-attr-val-1")
}

func initSpanEventAttributes(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString("span-event-attr", "span-event-attr-val")
}

func initSpanLinkAttributes(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString("span-link-attr", "span-link-attr-val")
}

func initMetricExemplarAttributes(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString("exemplar-attachment", "exemplar-attachment-value")
}

func initMetricAttributes1(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString("label-1", "label-value-1")
}

func initMetricAttributes12(dest pcommon.Map) {
	initMetricAttributes1(dest)
	dest.InsertString(testLabelKey2, testLabelValue2)
}

func initMetricAttributes13(dest pcommon.Map) {
	initMetricAttributes1(dest)
	dest.InsertString("label-3", "label-value-3")
}

func initMetricAttributes2(dest pcommon.Map) {
	dest.Clear()
	dest.InsertString(testLabelKey2, testLabelValue2)
}
