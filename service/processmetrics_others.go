//go:build !linux && !windows && !darwin

package service

import (
	"runtime"

	"go.uber.org/zap"
)

func registerProcessMetrics(srv *Service) error {
	srv.telemetrySettings.Logger.Warn(
		"Process metrics are disabled on this operating system",
		zap.String("os", runtime.GOOS),
	)
	return nil
}