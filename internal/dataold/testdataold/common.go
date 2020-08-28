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

package testdataold

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

var resourceAttributes1 = map[string]pdata.AttributeValue{"resource-attr": pdata.NewAttributeValueString("resource-attr-val-1")}

func initResource1(r pdata.Resource) {
	r.InitEmpty()
	initResourceAttributes1(r.Attributes())
}

func generateOtlpResource1() *otlpresource.Resource {
	return &otlpresource.Resource{
		Attributes: generateOtlpResourceAttributes1(),
	}
}

func initResourceAttributes1(dest pdata.AttributeMap) {
	dest.InitFromMap(resourceAttributes1)
}

func generateOtlpResourceAttributes1() []*otlpcommon.KeyValue {
	return []*otlpcommon.KeyValue{
		{
			Key:   "resource-attr",
			Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "resource-attr-val-1"}},
		},
	}
}

func initMetricLabels1(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey1: testdata.TestLabelValue1})
}

func generateOtlpMetricLabels1() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey1,
			Value: testdata.TestLabelValue1,
		},
	}
}

func initMetricLabelValue1(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey: testdata.TestLabelValue1})
}

func generateOtlpMetricLabelValue1() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey,
			Value: testdata.TestLabelValue1,
		},
	}
}

func initMetricLabels12(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey1: testdata.TestLabelValue1, testdata.TestLabelKey2: testdata.TestLabelValue2}).Sort()
}

func generateOtlpMetricLabels12() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey1,
			Value: testdata.TestLabelValue1,
		},
		{
			Key:   testdata.TestLabelKey2,
			Value: testdata.TestLabelValue2,
		},
	}
}

func initMetricLabels13(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey1: testdata.TestLabelValue1, testdata.TestLabelKey3: testdata.TestLabelValue3}).Sort()
}

func generateOtlpMetricLabels13() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey1,
			Value: testdata.TestLabelValue1,
		},
		{
			Key:   testdata.TestLabelKey3,
			Value: testdata.TestLabelValue3,
		},
	}
}

func initMetricLabels2(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey2: testdata.TestLabelValue2})
}

func generateOtlpMetricLabels2() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey2,
			Value: testdata.TestLabelValue2,
		},
	}
}

func initMetricLabelValue2(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestLabelKey: testdata.TestLabelValue2})
}

func generateOtlpMetricLabelValue2() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestLabelKey,
			Value: testdata.TestLabelValue2,
		},
	}
}

func initMetricAttachment(dest pdata.StringMap) {
	dest.InitFromMap(map[string]string{testdata.TestAttachmentKey: testdata.TestAttachmentValue})
}

func generateOtlpMetricAttachment() []*otlpcommon.StringKeyValue {
	return []*otlpcommon.StringKeyValue{
		{
			Key:   testdata.TestAttachmentKey,
			Value: testdata.TestAttachmentValue,
		},
	}
}
