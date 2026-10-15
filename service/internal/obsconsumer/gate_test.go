// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"testing"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

func setGateForTest(t *testing.T, enabled bool) {
	testutil.SetFeatureGate(t, metadata.TelemetryNewPipelineTelemetryFeatureGate.ID(), enabled)
}
