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
	"math"
	"testing"
	"time"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func Test_transaction(t *testing.T) {
	// discoveredLabels contain labels prior to any processing
	discoveredLabels := labels.New(
		labels.Label{
			Name:  model.AddressLabel,
			Value: "address:8080",
		},
		labels.Label{
			Name:  model.MetricNameLabel,
			Value: "foo",
		},
		labels.Label{
			Name:  model.SchemeLabel,
			Value: "http",
		},
	)
	// processedLabels contain label values after processing (e.g. relabeling)
	processedLabels := labels.New(
		labels.Label{
			Name:  model.InstanceLabel,
			Value: "localhost:8080",
		},
	)

	ms := &metadataService{
		sm: &mockScrapeManager{targets: map[string][]*scrape.Target{
			"test": {scrape.NewTarget(processedLabels, discoveredLabels, nil)},
		}},
	}

	rID := config.NewID("prometheus")

	t.Run("Commit Without Adding", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, nomc, nil, testLogger)
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
	})

	t.Run("Rollback dose nothing", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, nomc, nil, testLogger)
		if got := tr.Rollback(); got != nil {
			t.Errorf("expecting nil from Rollback() but got err %v", got)
		}
	})

	badLabels := labels.Labels([]labels.Label{{Name: "foo", Value: "bar"}})
	t.Run("Add One No Target", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, nomc, nil, testLogger)
		if _, got := tr.Append(0, badLabels, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	jobNotFoundLb := labels.Labels([]labels.Label{
		{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test2"},
		{Name: "foo", Value: "bar"}})
	t.Run("Add One Job not found", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, nomc, nil, testLogger)
		if _, got := tr.Append(0, jobNotFoundLb, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	goodLabels := labels.Labels([]labels.Label{{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test"},
		{Name: "__name__", Value: "foo"}})
	t.Run("Add One Good", func(t *testing.T) {
		sink := new(consumertest.MetricsSink)
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, sink, nil, testLogger)
		if _, got := tr.Append(0, goodLabels, time.Now().Unix()*1000, 1.0); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		tr.metricBuilder.startTime = 1.0 // set to a non-zero value
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
		expectedNode, expectedResource := createNodeAndResource("test", "localhost:8080", "http")
		mds := sink.AllMetrics()
		if len(mds) != 1 {
			t.Fatalf("wanted one batch, got %v\n", sink.AllMetrics())
		}
		var ocmds []*agentmetricspb.ExportMetricsServiceRequest
		rms := mds[0].ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			ocmd := &agentmetricspb.ExportMetricsServiceRequest{}
			ocmd.Node, ocmd.Resource, ocmd.Metrics = internaldata.ResourceMetricsToOC(rms.At(i))
			ocmds = append(ocmds, ocmd)
		}
		require.Len(t, ocmds, 1)
		if !proto.Equal(ocmds[0].Node, expectedNode) {
			t.Errorf("generated node %v and expected node %v is different\n", ocmds[0].Node, expectedNode)
		}
		if !proto.Equal(ocmds[0].Resource, expectedResource) {
			t.Errorf("generated resource %v and expected resource %v is different\n", ocmds[0].Resource, expectedResource)
		}

		// TODO: re-enable this when handle unspecified OC type
		// assert.Len(t, ocmds[0].Metrics, 1)
	})

	t.Run("Error when start time is zero", func(t *testing.T) {
		sink := new(consumertest.MetricsSink)
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, sink, nil, testLogger)
		if _, got := tr.Append(0, goodLabels, time.Now().Unix()*1000, 1.0); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		tr.metricBuilder.startTime = 0 // zero value means the start time metric is missing
		got := tr.Commit()
		if got == nil {
			t.Error("expecting error from Commit() but got nil")
		} else if got.Error() != errNoStartTimeMetrics.Error() {
			t.Errorf("expected error %q but got %q", errNoStartTimeMetrics, got)
		}
	})

	t.Run("Drop NaN value", func(t *testing.T) {
		sink := new(consumertest.MetricsSink)
		tr := newTransaction(context.Background(), nil, true, "", rID, ms, sink, nil, testLogger)
		if _, got := tr.Append(0, goodLabels, time.Now().Unix()*1000, math.NaN()); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
		if len(sink.AllMetrics()) != 0 {
			t.Errorf("wanted nil, got %v\n", sink.AllMetrics())
		}
	})

}
