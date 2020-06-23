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
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/testbed/testbed"
)

var _ testbed.MetricsDualConsumer = (*TestHarness)(nil)

type TestHarness struct {
	t                  *testing.T
	metricSupplier     *metricSupplier
	metricIndex        *metricIndex
	sender             *dualSender
	currMD             data.MetricData
	diffConsumer       DiffConsumer
	outOfMetrics       bool
	allMetricsReceived chan struct{}
}

type DiffConsumer interface {
	Accept(string, []*MetricDiff)
}

func NewTestHarness(t *testing.T, s *metricSupplier, mi *metricIndex, ds *dualSender, diffConsumer DiffConsumer) *TestHarness {
	return &TestHarness{
		t:                  t,
		metricSupplier:     s,
		metricIndex:        mi,
		sender:             ds,
		diffConsumer:       diffConsumer,
		allMetricsReceived: make(chan struct{}),
	}
}

func (h *TestHarness) ConsumeMetrics(_ context.Context, pdm pdata.Metrics) error {
	h.doConsumeMetrics(pdatautil.MetricsToInternalMetrics(pdm))
	return nil
}

// old
func (h *TestHarness) ConsumeMetricsData(_ context.Context, cmd consumerdata.MetricsData) error {
	md := pdatautil.MetricsToInternalMetrics(pdatautil.MetricsFromMetricsData([]consumerdata.MetricsData{cmd}))
	h.doConsumeMetrics(md)
	return nil
}

func (h *TestHarness) doConsumeMetrics(md data.MetricData) {
	h.compare(&md)
	if h.metricIndex.allSeen() {
		close(h.allMetricsReceived)
	}
	if !h.outOfMetrics {
		h.sendNextMetric()
	}
}

func (h *TestHarness) compare(md *data.MetricData) {
	pdms := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	var diffs []*MetricDiff
	for i := 0; i < pdms.Len(); i++ {
		pdmRecd := pdms.At(i)
		metricName := pdmRecd.MetricDescriptor().Name()
		metric, found := h.metricIndex.lookup(metricName)
		if !found {
			h.diffConsumer.Accept(metricName, []*MetricDiff{{
				ExpectedValue: metricName,
				Msg:           "Metric name not found in index",
			}})
		}
		if !metric.seen {
			metric.seen = true
			sent := metric.md
			pdmExpected := sent.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
			diffs = DiffMetric(
				diffs,
				pdmExpected,
				pdmRecd,
			)
			h.diffConsumer.Accept(metricName, diffs)
		}
	}
}

func (h *TestHarness) sendNextMetric() {
	log.Println("sending first metric")
	h.currMD, h.outOfMetrics = h.metricSupplier.nextMetricData()
	if h.outOfMetrics {
		log.Println("outOfMetrics")
		return
	}

	err := h.sender.send(h.currMD)
	require.NoError(h.t, err)
}

func metricNames(md *data.MetricData) (out []string) {
	rms := md.ResourceMetrics()
	if rms.Len() > 1 {
		panic("rms.Len() > 1")
	}
	ilm := rms.At(0).InstrumentationLibraryMetrics()
	if ilm.Len() > 1 {
		panic("ilm.Len() > 1")
	}
	metrics := ilm.At(0).Metrics()
	for i := 0; i < metrics.Len(); i++ {
		out = append(out, metrics.At(i).MetricDescriptor().Name())
	}
	return out
}
