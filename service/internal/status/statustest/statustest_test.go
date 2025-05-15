// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statustest // import "go.opentelemetry.io/collector/service/internal/status/statustest"

import "testing"

func TestNopStatusReporter(*testing.T) {
	nop := NewNopStatusReporter()
	nop.ReportOKIfStarting(nil)
	nop.ReportStatus(nil, nil)
}
