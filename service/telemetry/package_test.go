// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"testing"
)

func TestMain(m *testing.M) {
	// TODO: Remove this once the following PR is merged:
	// https://github.com/open-telemetry/opentelemetry-collector/pull/9241
	//goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
}
