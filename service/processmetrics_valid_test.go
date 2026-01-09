// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux || windows || darwin

package service // import "go.opentelemetry.io/collector/service"

import (
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestRegisterProcessMetrics_DoesNotPanic(t *testing.T) {
	srv := &Service{
		telemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// We don't assert on the error because it may depend on the runtime environment.
	if err := registerProcessMetrics(srv); err != nil {
		t.Logf("registerProcessMetrics returned error: %v", err)
	}
}
