//go:build linux || windows || darwin

package service

import "go.opentelemetry.io/collector/service/internal/proctelemetry"

func registerProcessMetrics(srv *Service) error {
	return proctelemetry.RegisterProcessMetrics(srv.telemetrySettings)
}