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

func initMetricExemplarAttributes(dest pcommon.Map) {
	dest.PutStr("exemplar-attachment", "exemplar-attachment-value")
}

func initMetricAttributes1(dest pcommon.Map) {
	dest.PutStr("label-1", "label-value-1")
}

func initMetricAttributes2(dest pcommon.Map) {
	dest.PutStr("label-2", "label-value-2")
}

func initMetricAttributes12(dest pcommon.Map) {
	initMetricAttributes1(dest)
	initMetricAttributes2(dest)
}

func initMetricAttributes13(dest pcommon.Map) {
	initMetricAttributes1(dest)
	dest.PutStr("label-3", "label-value-3")
}
