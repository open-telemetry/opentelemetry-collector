// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"testing"

	"go.uber.org/goleak"
)

// cspell:ignore gopkg, natefinch
func TestMain(m *testing.M) {
	// goleak.VerifyTestMain(m)
	// Ignore lumberjack millRun goroutine in all tests
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("gopkg.in/natefinch/lumberjack%2ev2.(*Logger).millRun"))
}
