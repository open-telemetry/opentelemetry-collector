// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

const defaultConcurrency = 100

type ConcurrencySettings struct {
	// Concurrency is a limit on the number of goroutines created for exporting batches.
	Concurrency int `mapstructure:"concurrency"`
}

func NewDefaultConcurrencySettings() *ConcurrencySettings {
	return &ConcurrencySettings{
		Concurrency: defaultConcurrency,
	}
}

func (cCfg *ConcurrencySettings) Validate() error {
	if cCfg.Concurrency <= 0 {
		return errors.New("number of concurrent senders must be positive")
	}

	return nil
}

// concurrencySender is a component that limits the number of RPC's that can take place concurrently.
// When used with the batchSender it can also send multiple batches concurrently if the concurrency limit has not been reached.
type concurrencySender struct {
	baseRequestSender
	cfg    ConcurrencySettings
	logger *zap.Logger
	sem    *semaphore.Weighted
}

// newConcurrencySender returns a new concurrency consumer component.
func newConcurrencySender(cfg ConcurrencySettings, set exporter.Settings) *concurrencySender {
	cs := &concurrencySender{
		cfg:    cfg,
		logger: set.Logger,
		sem:    semaphore.NewWeighted(int64(cfg.Concurrency)),
	}
	return cs
}

func (cs *concurrencySender) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (cs *concurrencySender) send(ctx context.Context, reqs ...Request) error {
	var errs error
	var wg sync.WaitGroup
	for _, req := range reqs {
		err := cs.sem.Acquire(ctx, int64(1))
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(r Request) {
			defer cs.sem.Release(int64(1))
			defer wg.Done()
			err := cs.nextSender.send(ctx, r)
			errs = multierr.Append(errs, err)
		}(req)
	}

	wg.Wait()
	return errs
}
