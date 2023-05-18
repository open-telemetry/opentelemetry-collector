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
	cfg              *Config
	logger           *zap.Logger
	ballast          []byte
	ballastSizeBytes uint64
	getTotalMem      func() (uint64, error)
}

func (m *memoryBallast) Start(_ context.Context, _ component.Host) error {
	// absolute value supersedes percentage setting
	if m.cfg.SizeMiB > 0 {
		m.ballastSizeBytes = m.cfg.SizeMiB * megaBytes
	} else {
		totalMemory, err := m.getTotalMem()
		if err != nil {
			return err
		}
		ballastPercentage := m.cfg.SizeInPercentage
		m.ballastSizeBytes = ballastPercentage * totalMemory / 100
	}

	if m.ballastSizeBytes > 0 {
		m.ballast = make([]byte, m.ballastSizeBytes)
	}

	m.logger.Info("Setting memory ballast", zap.Uint32("MiBs", uint32(m.ballastSizeBytes/megaBytes)))

	return nil
}

func (m *memoryBallast) Shutdown(_ context.Context) error {
	m.ballast = nil
	return nil
}

func newMemoryBallast(cfg *Config, logger *zap.Logger, getTotalMem func() (uint64, error)) *memoryBallast {
	return &memoryBallast{
		cfg:         cfg,
		logger:      logger,
		getTotalMem: getTotalMem,
	}
}

// GetBallastSize returns the current ballast memory setting in bytes
func (m *memoryBallast) GetBallastSize() uint64 {
	return m.ballastSizeBytes
}
