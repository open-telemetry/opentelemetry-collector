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

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
)

// pipelineSpec is designed for easy definition of expected pipeline structure
// It represents the structure of an individual pipeline, including instanced connectors
type pipelineSpec struct {
	id           component.ID
	receiverIDs  map[int64]component.ID
	processorIDs map[int64]component.ID
	exporterIDs  map[int64]component.ID
}

func newPipelineSpec(id component.ID) *pipelineSpec {
	return &pipelineSpec{
		id:           id,
		receiverIDs:  make(map[int64]component.ID),
		processorIDs: make(map[int64]component.ID),
		exporterIDs:  make(map[int64]component.ID),
	}
}

func (ps *pipelineSpec) withExampleReceiver(name string) *pipelineSpec {
	rID := component.NewIDWithName(component.Type("examplereceiver"), name)
	ps.receiverIDs[newReceiverNodeID(ps.id.Type(), rID).ID()] = rID
	return ps
}

func (ps *pipelineSpec) withExampleProcessor(name string) *pipelineSpec {
	pID := component.NewIDWithName(component.Type("exampleprocessor"), name)
	ps.processorIDs[newProcessorNodeID(ps.id, pID).ID()] = pID
	return ps
}

func (ps *pipelineSpec) withExampleExporter(name string) *pipelineSpec {
	eID := component.NewIDWithName(component.Type("exampleexporter"), name)
	ps.exporterIDs[newExporterNodeID(ps.id.Type(), eID).ID()] = eID
	return ps
}

func (ps *pipelineSpec) withExampleConnectorAsReceiver(name string, fromType component.Type) *pipelineSpec {
	cID := component.NewIDWithName(component.Type("exampleconnector"), name)
	ps.receiverIDs[newConnectorNodeID(fromType, ps.id.Type(), cID).ID()] = cID
	return ps
}

func (ps *pipelineSpec) withExampleConnectorAsExporter(name string, toType component.Type) *pipelineSpec {
	cID := component.NewIDWithName(component.Type("exampleconnector"), name)
	ps.exporterIDs[newConnectorNodeID(ps.id.Type(), toType, cID).ID()] = cID
	return ps
}

type pipelineSpecSlice []*pipelineSpec

func (psSlice pipelineSpecSlice) toMap() map[component.ID]*pipelineSpec {
	psMap := make(map[component.ID]*pipelineSpec, len(psSlice))
	for _, ps := range psSlice {
		psMap[ps.id] = ps
	}
	return psMap
}

type statefulComponentPipeline struct {
	receivers  map[int64]testcomponents.StatefulComponent
	processors map[int64]testcomponents.StatefulComponent
	exporters  map[int64]testcomponents.StatefulComponent
}

// Extract the component from each node in the pipeline and make it available as StatefulComponent
func (pg *pipelineGraph) toStatefulComponentPipeline() *statefulComponentPipeline {
	statefulPipeline := &statefulComponentPipeline{
		receivers:  make(map[int64]testcomponents.StatefulComponent),
		processors: make(map[int64]testcomponents.StatefulComponent),
		exporters:  make(map[int64]testcomponents.StatefulComponent),
	}

	for _, r := range pg.receivers {
		switch c := r.(type) {
		case *receiverNode:
			statefulPipeline.receivers[c.ID()] = c.Component.(*testcomponents.ExampleReceiver)
		case *connectorNode:
			// connector needs to be unwrapped to access component as ExampleConnector
			switch ct := c.Component.(type) {
			case connectorConsumerTraces:
				statefulPipeline.receivers[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			case connectorConsumerMetrics:
				statefulPipeline.receivers[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			case connectorConsumerLogs:
				statefulPipeline.receivers[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			}
		}
	}

	for _, p := range pg.processors {
		pn := p.(*processorNode)
		statefulPipeline.processors[pn.ID()] = pn.Component.(*testcomponents.ExampleProcessor)
	}

	for _, e := range pg.exporters {
		switch c := e.(type) {
		case *exporterNode:
			statefulPipeline.exporters[c.ID()] = c.Component.(*testcomponents.ExampleExporter)
		case *connectorNode:
			// connector needs to be unwrapped to access component as ExampleConnector
			switch ct := c.Component.(type) {
			case connectorConsumerTraces:
				statefulPipeline.exporters[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			case connectorConsumerMetrics:
				statefulPipeline.exporters[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			case connectorConsumerLogs:
				statefulPipeline.exporters[c.ID()] = ct.Component.(*testcomponents.ExampleConnector)
			}
		}
	}

	return statefulPipeline
}

func TestConnectorPipelinesGraph(t *testing.T) {
	// Even though we have a graph of nodes, it is very important that the notion of
	// pipelines is fully respected. Therefore, test expectations are defined on a
	// per-pipeline basis. Validation is also focused on each pipeline, but the net
	// effect is that the entire graph is carefully validated.
	tests := []struct {
		name                string
		pipelines           pipelineSpecSlice
		pipelineConfigs     map[component.ID]*ConfigServicePipeline
		expectedPerExporter int // requires symmetry in pipelines
	}{
		// Same test cases as old test
		{
			name: "pipelines_simple.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_multi_proc.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_simple_no_proc.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_multi.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter("").withExampleExporter("1"),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter("").withExampleExporter("1"),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleProcessor("").withExampleProcessor("1").
					withExampleExporter("").withExampleExporter("1"),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Processors: []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
					Exporters:  []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_multi_no_proc.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleExporter("").withExampleExporter("1"),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleExporter("").withExampleExporter("1"),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").withExampleReceiver("1").
					withExampleExporter("").withExampleExporter("1"),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("metrics"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
				component.NewID("logs"): {
					Receivers: []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
					Exporters: []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_exporter_multi_pipeline.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "1")).
					withExampleReceiver("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "1")).
					withExampleReceiver("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "1")).
					withExampleReceiver("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("traces", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("metrics", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("logs", "1"): {
					Receivers: []component.ID{component.NewID("examplereceiver")},
					Exporters: []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		// New test cases involving connectors
		{
			name: "pipelines_conn_simple_traces.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_metrics.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeMetrics),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_simple_logs.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_fork_merge_traces.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("fork", component.DataTypeTraces),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "type0")).
					withExampleConnectorAsReceiver("fork", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeTraces),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "type1")).
					withExampleConnectorAsReceiver("fork", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeTraces),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "out")).
					withExampleConnectorAsReceiver("merge", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
				},
				component.NewIDWithName("traces", "type0"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("traces", "type1"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_metrics.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("fork", component.DataTypeMetrics),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "type0")).
					withExampleConnectorAsReceiver("fork", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeMetrics),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "type1")).
					withExampleConnectorAsReceiver("fork", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeMetrics),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "out")).
					withExampleConnectorAsReceiver("merge", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
				},
				component.NewIDWithName("metrics", "type0"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("metrics", "type1"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_fork_merge_logs.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("fork", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "type0")).
					withExampleConnectorAsReceiver("fork", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "type1")).
					withExampleConnectorAsReceiver("fork", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleConnectorAsExporter("merge", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "out")).
					withExampleConnectorAsReceiver("merge", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
				},
				component.NewIDWithName("logs", "type0"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("logs", "type1"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "fork")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
				},
				component.NewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.NewIDWithName("exampleconnector", "merge")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 2,
		},
		{
			name: "pipelines_conn_translate_from_traces.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeMetrics).
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_metrics.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces).
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_translate_from_logs.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces).
					withExampleConnectorAsExporter("", component.DataTypeMetrics),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "")).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "")).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewID("logs"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewID("metrics"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},
			expectedPerExporter: 1,
		},
		{
			name: "pipelines_conn_matrix.yaml",
			pipelines: pipelineSpecSlice{
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces).
					withExampleConnectorAsExporter("", component.DataTypeMetrics).
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces).
					withExampleConnectorAsExporter("", component.DataTypeMetrics).
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "in")).
					withExampleReceiver("").
					withExampleProcessor("").
					withExampleConnectorAsExporter("", component.DataTypeTraces).
					withExampleConnectorAsExporter("", component.DataTypeMetrics).
					withExampleConnectorAsExporter("", component.DataTypeLogs),
				newPipelineSpec(component.NewIDWithName(component.DataTypeTraces, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeMetrics, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
				newPipelineSpec(component.NewIDWithName(component.DataTypeLogs, "out")).
					withExampleConnectorAsReceiver("", component.DataTypeTraces).
					withExampleConnectorAsReceiver("", component.DataTypeMetrics).
					withExampleConnectorAsReceiver("", component.DataTypeLogs).
					withExampleProcessor("").
					withExampleExporter(""),
			},
			pipelineConfigs: map[component.ID]*ConfigServicePipeline{
				component.NewIDWithName("traces", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("metrics", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("logs", "in"): {
					Receivers:  []component.ID{component.NewID("examplereceiver")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleconnector")},
				},
				component.NewIDWithName("traces", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("metrics", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
				component.NewIDWithName("logs", "out"): {
					Receivers:  []component.ID{component.NewID("exampleconnector")},
					Processors: []component.ID{component.NewID("exampleprocessor")},
					Exporters:  []component.ID{component.NewID("exampleexporter")},
				},
			},

			expectedPerExporter: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Build the pipeline
			pipelinesInterface, err := buildPipelinesGraph(context.Background(), pipelinesSettings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				ReceiverFactories: map[component.Type]component.ReceiverFactory{
					testcomponents.ExampleReceiverFactory.Type(): testcomponents.ExampleReceiverFactory,
				},
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("examplereceiver"):              testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
					component.NewIDWithName("examplereceiver", "1"): testcomponents.ExampleReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorFactories: map[component.Type]component.ProcessorFactory{
					testcomponents.ExampleProcessorFactory.Type(): testcomponents.ExampleProcessorFactory,
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("exampleprocessor"):              testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleprocessor", "1"): testcomponents.ExampleProcessorFactory.CreateDefaultConfig(),
				},
				ExporterFactories: map[component.Type]component.ExporterFactory{
					testcomponents.ExampleExporterFactory.Type(): testcomponents.ExampleExporterFactory,
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("exampleexporter"):              testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleexporter", "1"): testcomponents.ExampleExporterFactory.CreateDefaultConfig(),
				},
				ConnectorFactories: map[component.Type]component.ConnectorFactory{
					testcomponents.ExampleConnectorFactory.Type(): testcomponents.ExampleConnectorFactory,
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("exampleconnector"):                  testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleconnector", "fork"):  testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("exampleconnector", "merge"): testcomponents.ExampleConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: test.pipelineConfigs,
			})
			require.NoError(t, err)

			allPipelines, ok := pipelinesInterface.(*pipelinesGraph)
			require.True(t, ok)

			assert.Equal(t, len(test.pipelines), len(allPipelines.pipelineGraphs))

			// The entire graph of components is started topologically
			assert.NoError(t, allPipelines.StartAll(context.Background(), componenttest.NewNopHost()))

			// Check each pipeline individually, ensuring that all components are started
			// and that they have observed no signals yet.
			for pipelineID, pipeSpec := range test.pipelines.toMap() {
				pipeline, ok := allPipelines.pipelineGraphs[pipelineID]
				require.True(t, ok, "expected to find pipeline: %s", pipelineID.String())

				require.Equal(t, len(pipeSpec.receiverIDs), len(pipeline.receivers))
				require.Equal(t, len(pipeSpec.processorIDs), len(pipeline.processors))
				require.Equal(t, len(pipeSpec.exporterIDs), len(pipeline.exporters))

				// The pipelineGraph is cumbersome to work with in this context because
				// several type assertions & switches are necessary in order to access the
				// validation functions included on the example components (Started, Stopped, etc)
				// This gets a representation where all components are testcomponent.StatefulComponent
				actualPipeline := pipeline.toStatefulComponentPipeline()

				for expNodeID := range pipeSpec.exporterIDs {
					exp, ok := actualPipeline.exporters[expNodeID]
					require.True(t, ok)
					require.True(t, exp.Started())
					require.Equal(t, 0, len(exp.RecallTraces()))
					require.Equal(t, 0, len(exp.RecallMetrics()))
					require.Equal(t, 0, len(exp.RecallLogs()))
				}

				for procNodeID := range pipeSpec.processorIDs {
					proc, ok := actualPipeline.processors[procNodeID]
					require.True(t, ok)
					require.True(t, proc.Started())
				}

				for rcvrNodeID := range pipeSpec.receiverIDs {
					rcvr, ok := actualPipeline.receivers[rcvrNodeID]
					require.True(t, ok)
					require.True(t, rcvr.Started())
				}
			}

			// Push data into the pipelines. The list of receivers is retrieved directly from the overall
			// component graph because we do not want to duplicate signal inputs to receivers that are
			// shared between pipelines. The `allReceivers` function also excludes connectors, which we do
			// not want to directly inject with signals.
			allReceivers := allPipelines.getReceivers()
			for _, rcvr := range allReceivers[component.DataTypeTraces] {
				tracesReceiver := rcvr.(*testcomponents.ExampleReceiver)
				assert.NoError(t, tracesReceiver.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
			}
			for _, rcvr := range allReceivers[component.DataTypeMetrics] {
				metricsReceiver := rcvr.(*testcomponents.ExampleReceiver)
				assert.NoError(t, metricsReceiver.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
			}
			for _, rcvr := range allReceivers[component.DataTypeLogs] {
				logsReceiver := rcvr.(*testcomponents.ExampleReceiver)
				assert.NoError(t, logsReceiver.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
			}

			// Shut down the entire component graph
			assert.NoError(t, allPipelines.ShutdownAll(context.Background()))

			// Check each pipeline individually, ensuring that all components are stopped.
			for pipelineID, pipeSpec := range test.pipelines.toMap() {
				pipeline, ok := allPipelines.pipelineGraphs[pipelineID]
				require.True(t, ok, "expected to find pipeline: %s", pipelineID.String())

				actualPipeline := pipeline.toStatefulComponentPipeline()

				for rcvrNodeID := range pipeSpec.receiverIDs {
					rcvr, ok := actualPipeline.receivers[rcvrNodeID]
					require.True(t, ok)
					require.True(t, rcvr.Stopped())
				}

				for procNodeID := range pipeSpec.processorIDs {
					proc, ok := actualPipeline.processors[procNodeID]
					require.True(t, ok)
					require.True(t, proc.Stopped())
				}

				for expNodeID := range pipeSpec.exporterIDs {
					exp, ok := actualPipeline.exporters[expNodeID]
					require.True(t, ok)
					require.True(t, exp.Stopped())
				}
			}

			// Get the list of exporters directly from the overall component graph. Like receivers,
			// exclude connectors and validate each exporter once regardless of sharing between pipelines.
			allExporters := allPipelines.GetExporters()
			for _, exp := range allExporters[component.DataTypeTraces] {
				tracesExporter := exp.(*testcomponents.ExampleExporter)
				assert.Equal(t, test.expectedPerExporter, len(tracesExporter.RecallTraces()))
				for i := 0; i < test.expectedPerExporter; i++ {
					assert.EqualValues(t, testdata.GenerateTraces(1), tracesExporter.RecallTraces()[0])
				}
			}
			for _, exp := range allExporters[component.DataTypeMetrics] {
				metricsExporter := exp.(*testcomponents.ExampleExporter)
				assert.Equal(t, test.expectedPerExporter, len(metricsExporter.RecallMetrics()))
				for i := 0; i < test.expectedPerExporter; i++ {
					assert.EqualValues(t, testdata.GenerateMetrics(1), metricsExporter.RecallMetrics()[0])
				}
			}
			for _, exp := range allExporters[component.DataTypeLogs] {
				logsExporter := exp.(*testcomponents.ExampleExporter)
				assert.Equal(t, test.expectedPerExporter, len(logsExporter.RecallLogs()))
				for i := 0; i < test.expectedPerExporter; i++ {
					assert.EqualValues(t, testdata.GenerateLogs(1), logsExporter.RecallLogs()[0])
				}
			}
		})
	}
}

func TestGraphBuildErrors(t *testing.T) {
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()
	nopConnectorFactory := componenttest.NewNopConnectorFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()
	badConnectorFactory := newBadConnectorFactory()

	tests := []struct {
		name     string
		settings pipelinesSettings
		expected string
	}{
		{
			name: "not_supported_exporter_logs",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" exporter, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_metrics",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" exporter, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_exporter_traces",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badExporterFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" exporter, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_logs",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" processor, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_metrics",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" processor, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_processor_traces",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("bf")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" processor, in pipeline \"traces\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_logs",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" receiver, in pipeline \"logs\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_metrics",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" receiver, in pipeline \"metrics\": telemetry type is not supported",
		},
		{
			name: "not_supported_receiver_traces",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("bf"): badReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "telemetry type is not supported",
			// TODO expected: "failed to create \"bf\" receiver, in pipeline \"traces\": telemetry type is not supported",
		},

		{
			name: "not_supported_connector_traces_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"traces\" pipelines to \"traces\" pipelines"
		},
		{
			name: "not_supported_connector_traces_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"traces\" pipelines to \"metrics\" pipelines"
		},
		{
			name: "not_supported_connector_traces_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"traces\" pipelines to \"logs\" pipelines"
		},
		{
			name: "not_supported_connector_metrics_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"metrics\" pipelines to \"traces\" pipelines"
		},
		{
			name: "not_supported_connector_metrics_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"metris\" pipelines to \"metris\" pipelines"
		},
		{
			name: "not_supported_connector_metrics_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"metrics\" pipelines to \"logs\" pipelines"
		},
		{
			name: "not_supported_connector_logs_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"logs\" pipelines to \"traces\" pipelines"
		},
		{
			name: "not_supported_connector_logs_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"logs\" pipelines to \"metrics\" pipelines"
		},
		{
			name: "not_supported_connector_logs_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("bf"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("bf")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers: []component.ID{component.NewID("bf")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector \"bf\" not used between valid pipeline types",
			// TODO expected "connector \"bf\" does not support connections from \"logs\" pipelines to \"logs\" pipelines"
		},
		{
			name: "not_allowed_conn_omit_recv_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_conn_omit_recv_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_conn_omit_recv_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_conn_omit_exp_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_conn_omit_exp_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_conn_omit_exp_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "conn")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "each connector must be used as both receiver and exporter",
			// TODO expected: "connector \"nop\\conn\" must be used as both receiver and exporter",
		},
		{
			name: "not_allowed_simple_cycle_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
				},
			},
			expected: "cycle detected: connector \"nop/conn\", processor \"nop\", connector \"nop/conn\"",
		},
		{
			name: "not_allowed_simple_cycle_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
				},
			},
			expected: "cycle detected: connector \"nop/conn\", processor \"nop\", connector \"nop/conn\"",
		},
		{
			name: "not_allowed_simple_cycle_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
				},
			},
			expected: "cycle detected: connector \"nop/conn\", processor \"nop\", connector \"nop/conn\"",
		},
		{
			name: "not_allowed_deep_cycle_traces.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("traces", "1"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn1")},
					},
					component.NewIDWithName("traces", "2"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn1")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn2"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn2")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "cycle detected: connector \"nop/conn\", processor \"nop\", processor \"nop\", connector \"nop/conn1\", connector \"nop/conn\"",
			// TODO rebuild cycle in order
		},
		{
			name: "not_allowed_deep_cycle_metrics.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("metrics", "in"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("metrics", "1"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn1")},
					},
					component.NewIDWithName("metrics", "2"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn1")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn2"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("metrics", "out"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn2")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "cycle detected: processor \"nop\", connector \"nop/conn\", processor \"nop\", connector \"nop/conn1\", processor \"nop\"",
			// TODO rebuild cycle in order
		},
		{
			name: "not_allowed_deep_cycle_logs.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "conn"):  nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn1"): nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "conn2"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("logs", "in"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("logs", "1"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn1")},
					},
					component.NewIDWithName("logs", "2"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn1")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "conn2"), component.NewIDWithName("nop", "conn")},
					},
					component.NewIDWithName("logs", "out"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "conn2")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "cycle detected: connector \"nop/conn1\", processor \"nop\", processor \"nop\", connector \"nop/conn\", connector \"nop/conn1\"",
			// TODO rebuild cycle in order
		},
		{
			name: "not_allowed_deep_cycle_multi_signal.yaml",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopExporterFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "fork"):      nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "count"):     nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "forkagain"): nopConnectorFactory.CreateDefaultConfig(),
					component.NewIDWithName("nop", "rawlog"):    nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "fork")},
					},
					component.NewIDWithName("traces", "copy1"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "fork")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "count")},
					},
					component.NewIDWithName("traces", "copy2"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "fork")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "forkagain")},
					},
					component.NewIDWithName("traces", "copy2a"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "forkagain")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "count")},
					},
					component.NewIDWithName("traces", "copy2b"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "forkagain")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "rawlog")},
					},
					component.NewIDWithName("metrics", "count"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "count")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
					component.NewIDWithName("logs", "raw"): {
						Receivers:  []component.ID{component.NewIDWithName("nop", "rawlog")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewIDWithName("nop", "fork")}, // cannot loop back to "nop/fork"
					},
				},
			},
			expected: "cycle detected: processor \"nop\", processor \"nop\", connector \"nop/fork\", connector \"nop/forkagain\", processor \"nop\", connector \"nop/rawlog\", processor \"nop\"",
		},
		{
			name: "unknown_exporter_config",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
					},
				},
			},
			expected: "exporter \"nop/1\" is not configured",
		},
		{
			name: "unknown_exporter_factory",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("traces"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("unknown")},
					},
				},
			},
			expected: "exporter factory not available for: \"unknown\"",
		},
		{
			name: "unknown_processor_config",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "processor \"nop/1\" is not configured",
		},
		{
			name: "unknown_processor_factory",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ProcessorConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopProcessorFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("metrics"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("unknown")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "processor factory not available for: \"unknown\"",
		},
		{
			name: "unknown_receiver_config",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("nop"), component.NewIDWithName("nop", "1")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "receiver \"nop/1\" is not configured",
		},
		{
			name: "unknown_receiver_factory",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewID("logs"): {
						Receivers: []component.ID{component.NewID("unknown")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "receiver factory not available for: \"unknown\"",
		},
		{
			name: "unknown_connector_config",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewIDWithName("nop", "0"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewIDWithName("nop", "0"), component.NewIDWithName("nop", "1")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewIDWithName("nop", "0"), component.NewIDWithName("nop", "1")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			// Since nop/1 is neither a defined receiver or connector, expect "receiver" in error message
			expected: "receiver \"nop/1\" is not configured",
		},
		{
			name: "unknown_connector_factory",
			settings: pipelinesSettings{
				ReceiverConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ExporterConfigs: map[component.ID]component.Config{
					component.NewID("nop"): nopReceiverFactory.CreateDefaultConfig(),
				},
				ConnectorConfigs: map[component.ID]component.Config{
					component.NewID("unknown"): nopConnectorFactory.CreateDefaultConfig(),
				},
				PipelineConfigs: map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName("traces", "in"): {
						Receivers: []component.ID{component.NewID("nop")},
						Exporters: []component.ID{component.NewID("unknown")},
					},
					component.NewIDWithName("traces", "out"): {
						Receivers: []component.ID{component.NewID("unknown")},
						Exporters: []component.ID{component.NewID("nop")},
					},
				},
			},
			expected: "connector factory not available for: \"unknown\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := test.settings
			set.BuildInfo = component.NewDefaultBuildInfo()
			set.Telemetry = componenttest.NewNopTelemetrySettings()
			set.ReceiverFactories = map[component.Type]component.ReceiverFactory{
				nopReceiverFactory.Type(): nopReceiverFactory,
				badReceiverFactory.Type(): badReceiverFactory,
			}
			set.ProcessorFactories = map[component.Type]component.ProcessorFactory{
				nopProcessorFactory.Type(): nopProcessorFactory,
				badProcessorFactory.Type(): badProcessorFactory,
			}
			set.ExporterFactories = map[component.Type]component.ExporterFactory{
				nopExporterFactory.Type(): nopExporterFactory,
				badExporterFactory.Type(): badExporterFactory,
			}
			set.ConnectorFactories = map[component.Type]component.ConnectorFactory{
				nopConnectorFactory.Type(): nopConnectorFactory,
				badConnectorFactory.Type(): badConnectorFactory,
			}

			_, err := buildPipelinesGraph(context.Background(), set)
			assert.EqualError(t, err, test.expected)
		})
	}
}

// // This includes all tests from the previous implmentation, plus a new one
// // relevant only to the new graph-based implementation.
func TestGraphFailToStartAndShutdown(t *testing.T) {
	errReceiverFactory := newErrReceiverFactory()
	errProcessorFactory := newErrProcessorFactory()
	errExporterFactory := newErrExporterFactory()
	errConnectorFactory := newErrConnectorFactory()
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()
	nopConnectorFactory := componenttest.NewNopConnectorFactory()

	set := pipelinesSettings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverFactories: map[component.Type]component.ReceiverFactory{
			nopReceiverFactory.Type(): nopReceiverFactory,
			errReceiverFactory.Type(): errReceiverFactory,
		},
		ReceiverConfigs: map[component.ID]component.Config{
			component.NewID(nopReceiverFactory.Type()): nopReceiverFactory.CreateDefaultConfig(),
			component.NewID(errReceiverFactory.Type()): errReceiverFactory.CreateDefaultConfig(),
		},
		ProcessorFactories: map[component.Type]component.ProcessorFactory{
			nopProcessorFactory.Type(): nopProcessorFactory,
			errProcessorFactory.Type(): errProcessorFactory,
		},
		ProcessorConfigs: map[component.ID]component.Config{
			component.NewID(nopProcessorFactory.Type()): nopProcessorFactory.CreateDefaultConfig(),
			component.NewID(errProcessorFactory.Type()): errProcessorFactory.CreateDefaultConfig(),
		},
		ExporterFactories: map[component.Type]component.ExporterFactory{
			nopExporterFactory.Type(): nopExporterFactory,
			errExporterFactory.Type(): errExporterFactory,
		},
		ExporterConfigs: map[component.ID]component.Config{
			component.NewID(nopExporterFactory.Type()): nopExporterFactory.CreateDefaultConfig(),
			component.NewID(errExporterFactory.Type()): errExporterFactory.CreateDefaultConfig(),
		},
		ConnectorFactories: map[component.Type]component.ConnectorFactory{
			nopConnectorFactory.Type(): nopConnectorFactory,
			errConnectorFactory.Type(): errConnectorFactory,
		},
		ConnectorConfigs: map[component.ID]component.Config{
			component.NewIDWithName(nopConnectorFactory.Type(), "conn"): nopConnectorFactory.CreateDefaultConfig(),
			component.NewIDWithName(errConnectorFactory.Type(), "conn"): errConnectorFactory.CreateDefaultConfig(),
		},
	}

	dataTypes := []component.DataType{component.DataTypeTraces, component.DataTypeMetrics, component.DataTypeLogs}
	for _, dt := range dataTypes {
		t.Run(string(dt)+"/receiver", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop"), component.NewID("err")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := buildPipelinesGraph(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/processor", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop"), component.NewID("err")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := buildPipelinesGraph(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/exporter", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop"), component.NewID("err")},
				},
			}
			pipelines, err := buildPipelinesGraph(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		for _, dt2 := range dataTypes {
			t.Run(string(dt)+"/"+string(dt2)+"/connector", func(t *testing.T) {
				set.PipelineConfigs = map[component.ID]*ConfigServicePipeline{
					component.NewIDWithName(dt, "in"): {
						Receivers:  []component.ID{component.NewID("nop")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop"), component.NewIDWithName("err", "conn")},
					},
					component.NewIDWithName(dt2, "out"): {
						Receivers:  []component.ID{component.NewID("nop"), component.NewIDWithName("err", "conn")},
						Processors: []component.ID{component.NewID("nop")},
						Exporters:  []component.ID{component.NewID("nop")},
					},
				}
				pipelines, err := buildPipelinesGraph(context.Background(), set)
				assert.NoError(t, err)
				assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
				assert.Error(t, pipelines.ShutdownAll(context.Background()))
			})
		}
	}
}

func (g *pipelinesGraph) getReceivers() map[component.DataType]map[component.ID]component.Component {
	receiversMap := make(map[component.DataType]map[component.ID]component.Component)
	receiversMap[component.DataTypeTraces] = make(map[component.ID]component.Component)
	receiversMap[component.DataTypeMetrics] = make(map[component.ID]component.Component)
	receiversMap[component.DataTypeLogs] = make(map[component.ID]component.Component)

	for _, pg := range g.pipelineGraphs {
		for _, rcvrNode := range pg.receivers {
			rcvrOrConnNode := g.componentGraph.Node(rcvrNode.ID())
			rcvrNode, ok := rcvrOrConnNode.(*receiverNode)
			if !ok {
				continue
			}
			receiversMap[rcvrNode.pipelineType][rcvrNode.componentID] = rcvrNode.Component
		}
	}
	return receiversMap
}
