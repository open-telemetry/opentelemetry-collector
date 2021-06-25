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

package resourceprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var (
	cfg = &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		AttributesActions: []processorhelper.ActionKeyValue{
			{Key: "cloud.availability_zone", Value: "zone-1", Action: processorhelper.UPSERT},
			{Key: "k8s.cluster.name", FromAttribute: "k8s-cluster", Action: processorhelper.INSERT},
			{Key: "redundant-attribute", Action: processorhelper.DELETE},
		},
	}
)

func TestResourceProcessorAttributesUpsert(t *testing.T) {
	tests := []struct {
		name             string
		config           *Config
		sourceAttributes map[string]string
		wantAttributes   map[string]string
	}{
		{
			name:             "config_with_attributes_applied_on_nil_resource",
			config:           cfg,
			sourceAttributes: nil,
			wantAttributes: map[string]string{
				"cloud.availability_zone": "zone-1",
			},
		},
		{
			name:             "config_with_attributes_applied_on_empty_resource",
			config:           cfg,
			sourceAttributes: map[string]string{},
			wantAttributes: map[string]string{
				"cloud.availability_zone": "zone-1",
			},
		},
		{
			name:   "config_attributes_applied_on_existing_resource_attributes",
			config: cfg,
			sourceAttributes: map[string]string{
				"cloud.availability_zone": "to-be-replaced",
				"k8s-cluster":             "test-cluster",
				"redundant-attribute":     "to-be-removed",
			},
			wantAttributes: map[string]string{
				"cloud.availability_zone": "zone-1",
				"k8s-cluster":             "test-cluster",
				"k8s.cluster.name":        "test-cluster",
			},
		},
		{
			name: "config_attributes_replacement",
			config: &Config{
				ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
				AttributesActions: []processorhelper.ActionKeyValue{
					{Key: "k8s.cluster.name", FromAttribute: "k8s-cluster", Action: processorhelper.INSERT},
					{Key: "k8s-cluster", Action: processorhelper.DELETE},
				},
			},
			sourceAttributes: map[string]string{
				"k8s-cluster": "test-cluster",
			},
			wantAttributes: map[string]string{
				"k8s.cluster.name": "test-cluster",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test trace consumer
			ttn := new(consumertest.TracesSink)

			factory := NewFactory()
			rtp, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), tt.config, ttn)
			require.NoError(t, err)
			assert.True(t, rtp.Capabilities().MutatesData)

			sourceTraceData := generateTraceData(tt.sourceAttributes)
			wantTraceData := generateTraceData(tt.wantAttributes)
			err = rtp.ConsumeTraces(context.Background(), sourceTraceData)
			require.NoError(t, err)
			traces := ttn.AllTraces()
			require.Len(t, traces, 1)
			traces[0].ResourceSpans().At(0).Resource().Attributes().Sort()
			assert.EqualValues(t, wantTraceData, traces[0])

			// Test metrics consumer
			tmn := new(consumertest.MetricsSink)
			rmp, err := factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), tt.config, tmn)
			require.NoError(t, err)
			assert.True(t, rtp.Capabilities().MutatesData)

			sourceMetricData := generateMetricData(tt.sourceAttributes)
			wantMetricData := generateMetricData(tt.wantAttributes)
			err = rmp.ConsumeMetrics(context.Background(), sourceMetricData)
			require.NoError(t, err)
			metrics := tmn.AllMetrics()
			require.Len(t, metrics, 1)
			metrics[0].ResourceMetrics().At(0).Resource().Attributes().Sort()
			assert.EqualValues(t, wantMetricData, metrics[0])

			// Test logs consumer
			tln := new(consumertest.LogsSink)
			rlp, err := factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), tt.config, tln)
			require.NoError(t, err)
			assert.True(t, rtp.Capabilities().MutatesData)

			sourceLogData := generateLogData(tt.sourceAttributes)
			wantLogData := generateLogData(tt.wantAttributes)
			err = rlp.ConsumeLogs(context.Background(), sourceLogData)
			require.NoError(t, err)
			logs := tln.AllLogs()
			require.Len(t, logs, 1)
			logs[0].ResourceLogs().At(0).Resource().Attributes().Sort()
			assert.EqualValues(t, wantLogData, logs[0])
		})
	}
}

func TestResourceProcessorError(t *testing.T) {
	badCfg := &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		AttributesActions: nil,
	}

	// Test traces consumer
	factory := NewFactory()
	rtp, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), badCfg, consumertest.NewNop())
	require.Error(t, err)
	require.Nil(t, rtp)

	// Test metrics consumer
	rmp, err := factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), badCfg, consumertest.NewNop())
	require.Error(t, err)
	require.Nil(t, rmp)

	// Test logs consumer
	rlp, err := factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), badCfg, consumertest.NewNop())
	require.Error(t, err)
	require.Nil(t, rlp)
}

func generateTraceData(attributes map[string]string) pdata.Traces {
	td := testdata.GenerateTracesOneSpanNoResource()
	if attributes == nil {
		return td
	}
	resource := td.ResourceSpans().At(0).Resource()
	for k, v := range attributes {
		resource.Attributes().InsertString(k, v)
	}
	resource.Attributes().Sort()
	return td
}

func generateMetricData(attributes map[string]string) pdata.Metrics {
	md := testdata.GenerateMetricsOneMetricNoResource()
	if attributes == nil {
		return md
	}
	resource := md.ResourceMetrics().At(0).Resource()
	for k, v := range attributes {
		resource.Attributes().InsertString(k, v)
	}
	resource.Attributes().Sort()
	return md
}

func generateLogData(attributes map[string]string) pdata.Logs {
	ld := testdata.GenerateLogsOneLogRecordNoResource()
	if attributes == nil {
		return ld
	}
	resource := ld.ResourceLogs().At(0).Resource()
	for k, v := range attributes {
		resource.Attributes().InsertString(k, v)
	}
	resource.Attributes().Sort()
	return ld
}
