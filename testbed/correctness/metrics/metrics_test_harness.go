// Copyright The OpenTelemetry Authors
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

package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testbed/testbed"
)

// testHarness listens for datapoints from the receiver to which it is attached
// and when it receives one, it compares it to the datapoint that was previously
// sent out. It then sends the next datapoint, if there is one.
type testHarness struct {
	t                  *testing.T
	metricSupplier     *metricSupplier
	metricIndex        *metricsReceivedIndex
	sender             testbed.MetricDataSender
	currPDM            pdata.Metrics
	diffConsumer       diffConsumer
	outOfMetrics       bool
	allMetricsReceived chan struct{}
}

type diffConsumer interface {
	accept(string, []*MetricDiff)
}

func newTestHarness(
	t *testing.T,
	s *metricSupplier,
	mi *metricsReceivedIndex,
	ds testbed.MetricDataSender,
	diffConsumer diffConsumer,
) *testHarness {
	return &testHarness{
		t:                  t,
		metricSupplier:     s,
		metricIndex:        mi,
		sender:             ds,
		diffConsumer:       diffConsumer,
		allMetricsReceived: make(chan struct{}),
	}
}

func (h *testHarness) ConsumeMetrics(_ context.Context, pdm pdata.Metrics) error {
	h.compare(pdm)
	if h.metricIndex.allReceived() {
		close(h.allMetricsReceived)
	}
	if !h.outOfMetrics {
		h.sendNextMetric()
	}
	return nil
}

func (h *testHarness) compare(pdm pdata.Metrics) {
	pdms := pdm.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	var diffs []*MetricDiff
	for i := 0; i < pdms.Len(); i++ {
		pdmRecd := pdms.At(i)
		metricName := pdmRecd.Name()
		metric, found := h.metricIndex.lookup(metricName)
		if !found {
			h.diffConsumer.accept(metricName, []*MetricDiff{{
				ExpectedValue: metricName,
				Msg:           "Metric name not found in index",
			}})
		}
		if !metric.received {
			metric.received = true
			sent := metric.pdm
			pdmExpected := sent.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
			diffs = DiffMetric(
				diffs,
				pdmExpected,
				pdmRecd,
			)
			h.diffConsumer.accept(metricName, diffs)
		}
	}
}

func (h *testHarness) sendNextMetric() {
	h.currPDM, h.outOfMetrics = h.metricSupplier.nextMetrics()
	if h.outOfMetrics {
		return
	}
	err := h.sender.ConsumeMetrics(context.Background(), h.currPDM)
	require.NoError(h.t, err)
}
