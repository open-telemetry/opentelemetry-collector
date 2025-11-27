// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil // import "go.opentelemetry.io/collector/internal/testutil"

import (
	"os"
	"testing"
)

// SkipMemoryBench will skip memory benchmarks on CI, as we currently only
// monitor duration.
func SkipMemoryBench(b *testing.B) {
	if os.Getenv("MEMBENCH") == "" {
		b.Skip("Skipping since the 'MEMBENCH' environment variable was not set")
	}
}
