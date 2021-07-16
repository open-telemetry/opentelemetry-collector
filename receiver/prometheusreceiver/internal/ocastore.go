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
	"errors"
	"sync/atomic"

	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	runningStateInit = iota
	runningStateReady
	runningStateStop
)

var idSeq int64
var noop = &noopAppender{}

// OcaStore translates Prometheus scraping diffs into OpenCensus format.
type OcaStore struct {
	ctx context.Context

	running              int32 // access atomically
	sink                 consumer.Metrics
	mc                   *metadataService
	jobsMap              *JobsMap
	useStartTimeMetric   bool
	startTimeMetricRegex string
	receiverID           config.ComponentID
	externalLabels       labels.Labels

	logger         *zap.Logger
	stalenessStore *stalenessStore
}

// NewOcaStore returns an ocaStore instance, which can be acted as prometheus' scrape.Appendable
func NewOcaStore(
	ctx context.Context,
	sink consumer.Metrics,
	logger *zap.Logger,
	jobsMap *JobsMap,
	useStartTimeMetric bool,
	startTimeMetricRegex string,
	receiverID config.ComponentID,
	externalLabels labels.Labels) *OcaStore {
	return &OcaStore{
		running:              runningStateInit,
		ctx:                  ctx,
		sink:                 sink,
		logger:               logger,
		jobsMap:              jobsMap,
		useStartTimeMetric:   useStartTimeMetric,
		startTimeMetricRegex: startTimeMetricRegex,
		receiverID:           receiverID,
		externalLabels:       externalLabels,
		stalenessStore:       newStalenessStore(),
	}
}

// SetScrapeManager is used to config the underlying scrape.Manager as it's needed for OcaStore, otherwise OcaStore
// cannot accept any Appender() request
func (o *OcaStore) SetScrapeManager(scrapeManager *scrape.Manager) {
	if scrapeManager != nil && atomic.CompareAndSwapInt32(&o.running, runningStateInit, runningStateReady) {
		o.mc = &metadataService{sm: scrapeManager}
	}
}

func (o *OcaStore) Appender(context.Context) storage.Appender {
	state := atomic.LoadInt32(&o.running)
	if state == runningStateReady {
		// Firstly prepare the stalenessStore for a new scrape cyle.
		o.stalenessStore.refresh()

		return newTransaction(
			o.ctx,
			o.jobsMap,
			o.useStartTimeMetric,
			o.startTimeMetricRegex,
			o.receiverID,
			o.mc,
			o.sink,
			o.externalLabels,
			o.logger,
			o.stalenessStore,
		)
	} else if state == runningStateInit {
		panic("ScrapeManager is not set")
	}
	// instead of returning an error, return a dummy appender instead, otherwise it can trigger panic
	return noop
}

// Close OcaStore as well as the internal metadataService.
func (o *OcaStore) Close() {
	if atomic.CompareAndSwapInt32(&o.running, runningStateReady, runningStateStop) {
		o.mc.Close()
	}
}

// noopAppender, always return error on any operations
type noopAppender struct{}

var errAlreadyStopped = errors.New("already stopped")

func (*noopAppender) Append(uint64, labels.Labels, int64, float64) (uint64, error) {
	return 0, errAlreadyStopped
}

func (*noopAppender) AppendExemplar(ref uint64, l labels.Labels, e exemplar.Exemplar) (uint64, error) {
	return 0, errAlreadyStopped
}

func (*noopAppender) Commit() error {
	return errAlreadyStopped
}

func (*noopAppender) Rollback() error {
	return nil
}
