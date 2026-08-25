// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"
)

func FuzzUnmarshalMetrics(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		u := &JSONUnmarshaler{}
		_, _ = u.UnmarshalMetrics(data)
	})
}
