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

// SkipGCHeavyBench will skip GC-heavy benchmarks on CI.
// These benchmarks tend to be flaky with the current settings since garbage
// collection pauses can take ~50ms which is significant with the current benchmark times.
func SkipGCHeavyBench(b *testing.B) {
	if os.Getenv("GCHEAVYBENCH") == "" {
		b.Skip("Skipping since the 'GCHEAVYBENCH' environment variable was not set. See https://github.com/open-telemetry/opentelemetry-collector/issues/14257.")
	}
}
