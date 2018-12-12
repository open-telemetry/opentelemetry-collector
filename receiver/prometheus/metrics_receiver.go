// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"errors"
	"time"

	"github.com/census-instrumentation/opencensus-service/receiver"
)

// Receiver is the type used to handle metrics from OpenCensus exporters.
type Receiver struct {
	metricSink         receiver.MetricsReceiverSink
	metricBufferPeriod time.Duration
	metricBufferCount  int
}

// New creates a new prometheus.Receiver reference.
func New(ms receiver.MetricsReceiverSink, opts ...Option) (*Receiver, error) {
	if ms == nil {
		return nil, errors.New("needs a non-nil receiver.MetricsReceiverSink")
	}
	pr := &Receiver{metricSink: ms}
	for _, opt := range opts {
		opt.WithReceiver(pr)
	}
	return pr, nil
}

const receiverName = "prometheus"

// Export is the gRPC method that exports streamed metrics from
// OpenCensus-metricproto compatible libraries/applications to MetricSink.
func (pr *Receiver) Export() error {
	// TODO: scrape metrics from Prometheus endpoint and convert to OC metrics.
	return nil
}
