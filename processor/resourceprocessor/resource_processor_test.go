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
	ptest "github.com/open-telemetry/opentelemetry-collector/processor/processortest"
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

	ctc := &ptest.CachingTraceConsumer{}
	rtp := newResourceTraceProcessor(ctc, cfg)
	assert.True(t, rtp.GetCapabilities().MutatesConsumedData)

	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ctc.Data, want)
}

func TestTraceResourceProcessorEmpty(t *testing.T) {
	want := consumerdata.TraceData{
		Resource: resource2,
	}
	test := consumerdata.TraceData{
		Resource: resource2,
	}

	ctc := &ptest.CachingTraceConsumer{}
	rtp := newResourceTraceProcessor(ctc, emptyCfg)
	assert.False(t, rtp.GetCapabilities().MutatesConsumedData)

	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ctc.Data, want)
}

func TestTraceResourceProcessorNonEmptyIncomingResource(t *testing.T) {
	want := consumerdata.TraceData{
		Resource: mergedResource,
	}
	test := consumerdata.TraceData{
		Resource: resource2,
	}
	ctc := &ptest.CachingTraceConsumer{}
	rtp := newResourceTraceProcessor(ctc, cfg)
	rtp.ConsumeTraceData(context.Background(), test)
	assert.Equal(t, ctc.Data, want)
}

func TestMetricResourceProcessor(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: resource,
	}
	test := consumerdata.MetricsData{}

	cmc := &ptest.CachingMetricsConsumer{}
	rmp := newResourceMetricProcessor(cmc, cfg)
	assert.True(t, rmp.GetCapabilities().MutatesConsumedData)

	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, cmc.Data, want)
}

func TestMetricResourceProcessorEmpty(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: resource2,
	}
	test := consumerdata.MetricsData{
		Resource: resource2,
	}

	cmc := &ptest.CachingMetricsConsumer{}
	rmp := newResourceMetricProcessor(cmc, emptyCfg)
	assert.False(t, rmp.GetCapabilities().MutatesConsumedData)

	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, cmc.Data, want)
}

func TestMetricResourceProcessorNonEmptyIncomingResource(t *testing.T) {
	want := consumerdata.MetricsData{
		Resource: mergedResource,
	}
	test := consumerdata.MetricsData{
		Resource: resource2,
	}

	cmc := &ptest.CachingMetricsConsumer{}
	rmp := newResourceMetricProcessor(cmc, cfg)
	rmp.ConsumeMetricsData(context.Background(), test)
	assert.Equal(t, cmc.Data, want)
}

func TestMergeResourceWithNilLabels(t *testing.T) {
	resourceNilLabels := &resourcepb.Resource{Type: "host"}
	assert.Nil(t, resourceNilLabels.Labels)
	assert.Equal(t, mergeResource(nil, resourceNilLabels), &resourcepb.Resource{Type: "host", Labels: map[string]string{}})
}
