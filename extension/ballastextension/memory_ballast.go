package ballastextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

type memoryBallast struct {
}

func (m memoryBallast) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (m memoryBallast) Shutdown(_ context.Context) error {
	return nil
}

func newMemoryBallast(_ *Config) *memoryBallast {
	return &memoryBallast{}
}
