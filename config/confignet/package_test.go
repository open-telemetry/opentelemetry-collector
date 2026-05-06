// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// go-winio starts a background goroutine for I/O completion on Windows that outlives the test.
	goleak.VerifyTestMain(m, goleak.IgnoreAnyFunction("github.com/Microsoft/go-winio.ioCompletionProcessor"))
}
