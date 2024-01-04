// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ballastextension // import "go.opentelemetry.io/collector/extension/ballastextension"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

const megaBytes = 1024 * 1024

type memoryBallast struct {
	component.StartFunc
	ballast          []byte
	ballastSizeBytes uint64
}

func (m *memoryBallast) Shutdown(_ context.Context) error {
	m.ballast = nil
	return nil
}

func newMemoryBallast(cfg *Config, logger *zap.Logger, getTotalMem func() (uint64, error)) (*memoryBallast, error) {
	ballastSizeBytes, err := calculateBallastSizeBytes(cfg, getTotalMem)
	if err != nil {
		return nil, err
	}

	logger.Info("Setting memory ballast", zap.Uint32("MiBs", uint32(ballastSizeBytes/megaBytes)))

	return &memoryBallast{
		ballastSizeBytes: ballastSizeBytes,
		ballast:          make([]byte, ballastSizeBytes),
	}, nil
}

// calculateBallastSizeBytes calculates the ballast size in bytes based on the configuration.
// If an absolute value is set, it will be used. Otherwise, the percentage will be used.
func calculateBallastSizeBytes(cfg *Config, getTotalMem func() (uint64, error)) (uint64, error) {
	if cfg.SizeMiB > 0 {
		return cfg.SizeMiB * megaBytes, nil
	}
	totalMemory, err := getTotalMem()
	if err != nil {
		return 0, err
	}
	return cfg.SizeInPercentage * totalMemory / 100, nil
}

// GetBallastSize returns the current ballast memory setting in bytes
func (m *memoryBallast) GetBallastSize() uint64 {
	return m.ballastSizeBytes
}
