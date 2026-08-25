// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil // import "go.opentelemetry.io/collector/internal/testutil"

import (
	"os"
	"strings"
	"testing"
)

// SkipIfFIPSOnly will mark the passed test as skipped if GODEBUG=fips140=only is detected.
// If GODEBUG=fips140=on, go may call non-compliant algorithms and the test does not need to be skipped.
func SkipIfFIPSOnly(t *testing.T, msg string) {
	// NOTE: This only checks env var; at the time of writing fips140 can only be set via env
	// other GODEBUG settings can be set via embedded comments or in go.mod, we may need to account for this in the future.
	s := os.Getenv("GODEBUG")
	if strings.Contains(s, "fips140=only") {
		t.Skip("GODEBUG=fips140=only detected, skipping test:", msg)
	}
}
