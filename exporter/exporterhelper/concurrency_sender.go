// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

type ConcurrencySettings struct {
	Enabled    bool `mapstructure:"enabled"`
	NumSenders int  `mapstructure:"num_senders"`
}
// batchSender is a component that places requests into batches before passing them to the downstream senders.
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.MinSizeItems
// - cfg.FlushTimeout is elapsed since the timestamp when the previous batch was sent out.
// - concurrencyLimit is reached.
type concurrencySender struct {
	baseRequestSender
	cfg    ConcurrencySettings
	mu     sync.Mutex
	logger *zap.Logger
	sem    *semaphore.Weighted
}

// newBatchSender returns a new batch consumer component.
func newConcurrencySender(cfg ConcurrencySettings, set exporter.Settings) *concurrencySender {
	cs := &concurrencySender{
		cfg:                cfg,
		logger:             set.Logger,
		sem:                semaphore.NewWeighted(int64(cfg.NumSenders)),
	}
	return cs
}

func (cs *concurrencySender) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (cs *concurrencySender) send(ctx context.Context, reqs ...Request) error {
	var errs error
	fmt.Println("CONC SENDER NUM REQS")
	fmt.Println(len(reqs))
	var wg sync.WaitGroup
	for _, r := range reqs {
		cs.sem.Acquire(ctx, int64(1))

		wg.Add(1)
		go func() {
			err := cs.nextSender.send(ctx, r)
			errs = multierr.Append(errs, err)
			cs.sem.Release(int64(1))
			wg.Done()
		}()
	}
	wg.Wait()
	return errs
}
