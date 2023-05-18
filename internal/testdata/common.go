// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
