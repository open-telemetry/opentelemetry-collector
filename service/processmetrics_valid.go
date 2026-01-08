// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux || windows || darwin

package service // import "go.opentelemetry.io/collector/service"

import (
	"fmt"

	"go.opentelemetry.io/collector/service/internal/proctelemetry"
)

func registerProcessMetrics(srv *Service) error {
	if err := proctelemetry.RegisterProcessMetrics(srv.telemetrySettings); err != nil {
		return fmt.Errorf("failed to register process metrics: %w", err)
	}
	return nil
}
