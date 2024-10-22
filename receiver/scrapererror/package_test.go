// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
// Deprecated: [v0.111.0] Use /scraper/scrapererror instead.
package scrapererror

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
