// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"
)

func FuzzUnmarshalTraces(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		u := &JSONUnmarshaler{}
		//nolint: errcheck
		_, _ = u.UnmarshalTraces(data)
	})
}
