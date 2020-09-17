// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
)

const (
	runningStateInit = iota
	runningStateReady
	runningStateStop
)

var idSeq int64
var noop = &noopAppender{}

// OcaStore is an interface combines io.Closer and prometheus' scrape.Appendable
type OcaStore interface {
	storage.Appendable
	io.Closer
	SetScrapeManager(*scrape.Manager)
}

// OpenCensus Store for prometheus
type ocaStore struct {
	running              int32
	logger               *zap.Logger
	sink                 consumer.MetricsConsumer
	mc                   *mService
	once                 *sync.Once
	ctx                  context.Context
	jobsMap              *JobsMap
	useStartTimeMetric   bool
	startTimeMetricRegex string
	receiverName         string
}

// NewOcaStore returns an ocaStore instance, which can be acted as prometheus' scrape.Appendable
func NewOcaStore(ctx context.Context, sink consumer.MetricsConsumer, logger *zap.Logger, jobsMap *JobsMap, useStartTimeMetric bool, startTimeMetricRegex string, receiverName string) OcaStore {
	return &ocaStore{
		running:              runningStateInit,
		ctx:                  ctx,
		sink:                 sink,
		logger:               logger,
		once:                 &sync.Once{},
		jobsMap:              jobsMap,
		useStartTimeMetric:   useStartTimeMetric,
		startTimeMetricRegex: startTimeMetricRegex,
		receiverName:         receiverName,
	}
}

// SetScrapeManager is used to config the underlying scrape.Manager as it's needed for OcaStore, otherwise OcaStore
// cannot accept any Appender() request
func (o *ocaStore) SetScrapeManager(scrapeManager *scrape.Manager) {
	if scrapeManager != nil && atomic.CompareAndSwapInt32(&o.running, runningStateInit, runningStateReady) {
		o.mc = &mService{sm: scrapeManager}
	}
}

func (o *ocaStore) Appender(context.Context) storage.Appender {
	state := atomic.LoadInt32(&o.running)
	if state == runningStateReady {
		return newTransaction(o.ctx, o.jobsMap, o.useStartTimeMetric, o.startTimeMetricRegex, o.receiverName, o.mc, o.sink, o.logger)
	} else if state == runningStateInit {
		panic("ScrapeManager is not set")
	}
	// instead of returning an error, return a dummy appender instead, otherwise it can trigger panic
	return noop
}

func (o *ocaStore) Close() error {
	atomic.CompareAndSwapInt32(&o.running, runningStateReady, runningStateStop)
	return nil
}

// noopAppender, always return error on any operations
type noopAppender struct{}

func (*noopAppender) Add(labels.Labels, int64, float64) (uint64, error) {
	return 0, componenterror.ErrAlreadyStopped
}

func (*noopAppender) AddFast(uint64, int64, float64) error {
	return componenterror.ErrAlreadyStopped
}

func (*noopAppender) Commit() error {
	return componenterror.ErrAlreadyStopped
}

func (*noopAppender) Rollback() error {
	return nil
}
