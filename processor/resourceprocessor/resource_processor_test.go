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
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
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

	cfgWithEmptyResourceType = &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		ResourceType: "",
		Labels: map[string]string{
			"cloud.zone":       "zone-1",
			"k8s.cluster.name": "k8s-cluster",
			"host.name":        "k8s-node",
		},
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

func TestResourceProcessor(t *testing.T) {
	tests := []struct {
		name                string
		config              *Config
		mutatesConsumedData bool
		sourceResource      *resourcepb.Resource
		wantResource        *resourcepb.Resource
	}{
		{
			name:                "Config with empty resource type doesn't mutate resource type",
			config:              cfgWithEmptyResourceType,
			mutatesConsumedData: true,
			sourceResource: &resourcepb.Resource{
				Type: "original-type",
				Labels: map[string]string{
					"original-label": "original-value",
					"cloud.zone":     "will-be-overridden",
				},
			},
			wantResource: &resourcepb.Resource{
				Type: "original-type",
				Labels: map[string]string{
					"original-label":   "original-value",
					"cloud.zone":       "zone-1",
					"k8s.cluster.name": "k8s-cluster",
					"host.name":        "k8s-node",
				},
			},
		},
		{
			name:                "Config with empty resource type keeps nil resource",
			config:              cfgWithEmptyResourceType,
			mutatesConsumedData: true,
			sourceResource:      nil,
			wantResource:        nil,
		},
		{
			name:                "Consumed resource with nil labels",
			config:              cfgWithEmptyResourceType,
			mutatesConsumedData: true,
			sourceResource: &resourcepb.Resource{
				Type: "original-type",
			},
			wantResource: &resourcepb.Resource{
				Type: "original-type",
				Labels: map[string]string{
					"cloud.zone":       "zone-1",
					"k8s.cluster.name": "k8s-cluster",
					"host.name":        "k8s-node",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test trace consuner
			ttn := &testTraceConsumer{}
			rtp := newResourceTraceProcessor(ttn, tt.config)
			assert.Equal(t, tt.mutatesConsumedData, rtp.GetCapabilities().MutatesConsumedData)

			err := rtp.ConsumeTraceData(context.Background(), consumerdata.TraceData{
				Resource: tt.sourceResource,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantResource, ttn.td.Resource)

			// Test metrics consumer
			tmn := &testMetricsConsumer{}
			rmp := newResourceMetricProcessor(tmn, tt.config)
			assert.Equal(t, tt.mutatesConsumedData, rmp.GetCapabilities().MutatesConsumedData)

			err = rmp.ConsumeMetricsData(context.Background(), consumerdata.MetricsData{
				Resource: tt.sourceResource,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantResource, tmn.md.Resource)
		})
	}
}

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
