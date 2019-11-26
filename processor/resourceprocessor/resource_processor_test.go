// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourceprocessor

import (
	"context"
	"testing"

	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

var (
	cfg = &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		ResourceType: "host",
		Labels: map[string]string{
			"cloud.zone":       "zone-1",
			"k8s.cluster.name": "k8s-cluster",
			"host.name":        "k8s-node",
		},
	}

	emptyCfg = &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		ResourceType: "",
		Labels:       map[string]string{},
	}

	resource = &resourcepb.Resource{
		Type: "host",
		Labels: map[string]string{
			"cloud.zone":       "zone-1",
			"k8s.cluster.name": "k8s-cluster",
			"host.name":        "k8s-node",
		},
	}

	resource2 = &resourcepb.Resource{
		Type: "ht",
		Labels: map[string]string{
			"zone":    "zone-2",
			"cluster": "cluster-2",
			"host":    "node-2",
		},
	}

	mergedResource = &resourcepb.Resource{
		Type: "host",
		Labels: map[string]string{
			"cloud.zone":       "zone-1",
			"k8s.cluster.name": "k8s-cluster",
			"host.name":        "k8s-node",
			"zone":             "zone-2",
			"cluster":          "cluster-2",
			"host":             "node-2",
		},
	}
)

func TestTraceResourceProcessor(t *testing.T) {
	want := consumerdata.TraceData{
		Resource: resource,
	}
	test := consumerdata.TraceData{}

	ttn := &testTraceConsumer{}
	rtp := newResourceTraceProcessor(ttn, cfg)
	assert.True(t, rtp.GetCapabilities().MutatesConsumedData)

	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ttn.td, want)
}

func TestTraceResourceProcessorEmpty(t *testing.T) {
	want := consumerdata.TraceData{
		Resource: resource2,
	}
	test := consumerdata.TraceData{
		Resource: resource2,
	}

	ttn := &testTraceConsumer{}
	rtp := newResourceTraceProcessor(ttn, emptyCfg)
	assert.False(t, rtp.GetCapabilities().MutatesConsumedData)

	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ttn.td, want)
}

func TestTraceResourceProcessorNonEmptyIncomingResource(t *testing.T) {
	want := consumerdata.TraceData{
		Resource: mergedResource,
	}
	test := consumerdata.TraceData{
		Resource: resource2,
	}
	ttn := &testTraceConsumer{}
	rtp := newResourceTraceProcessor(ttn, cfg)
	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ttn.td, want)
}

func TestMetricResourceProcessor(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: resource,
	}
	test := consumerdata.MetricsData{}

	tmn := &testMetricsConsumer{}
	rmp := newResourceMetricProcessor(tmn, cfg)
	assert.True(t, rmp.GetCapabilities().MutatesConsumedData)

	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, tmn.md, want)
}

func TestMetricResourceProcessorEmpty(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: resource2,
	}
	test := consumerdata.MetricsData{
		Resource: resource2,
	}

	tmn := &testMetricsConsumer{}
	rmp := newResourceMetricProcessor(tmn, emptyCfg)
	assert.False(t, rmp.GetCapabilities().MutatesConsumedData)

	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, tmn.md, want)
}

func TestMetricResourceProcessorNonEmptyIncomingResource(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: mergedResource,
	}
	test := consumerdata.MetricsData{
		Resource: resource2,
	}

	tmn := &testMetricsConsumer{}
	rmp := newResourceMetricProcessor(tmn, cfg)
	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, tmn.md, want)
}

func TestMergeResourceWithNilLabels(t *testing.T) {
	resourceNilLabels := &resourcepb.Resource{Type: "host"}
	assert.Nil(t, resourceNilLabels.Labels)
	assert.Equal(t, mergeResource(nil, resourceNilLabels), &resourcepb.Resource{Type: "host", Labels: map[string]string{}})
}

type testTraceConsumer struct {
	td consumerdata.TraceData
}

func (ttn *testTraceConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ttn.td = td
	return nil
}

type testMetricsConsumer struct {
	md consumerdata.MetricsData
}

func (tmn *testMetricsConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	tmn.md = md
	return nil
}
