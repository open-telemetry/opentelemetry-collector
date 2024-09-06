// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"testing"
)

func FuzzUnmarshalProfiles(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		u := &JSONUnmarshaler{}
		//nolint: errcheck
		_, _ = u.UnmarshalProfiles(data)
	})
}
