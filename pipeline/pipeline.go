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

package pipeline // import "go.opentelemetry.io/collector/pipeline"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/multierr"
)

var _ component.Host = (*host)(nil)

// host is a component.Host for a single pipeline.
// It has no extensions and a single exporter.
type host struct {
	// factories available for components.
	factories component.Factories
	// datatype is the telemetry signal type for the pipeline.
	datatype config.DataType
	// exporterID is the component ID for the pipeline's exporter.
	exporterID config.ComponentID
	// exporter is the exporter on this pipeline.
	exporter component.Exporter
	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// ReportFatalError implements the component.Host interface.
func (h *host) ReportFatalError(err error) {
	h.asyncErrorChannel <- err
}

// GetFactory implements the component.Host interface.
func (h *host) GetFactory(kind component.Kind, componentType config.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return h.factories.Receivers[componentType]
	case component.KindExtension:
		return h.factories.Extensions[componentType]
	case component.KindExporter:
		return h.factories.Exporters[componentType]
	case component.KindProcessor:
		return h.factories.Processors[componentType]
	}

	return nil
}

// GetExtensions implements the component.Host interface.
func (*host) GetExtensions() map[config.ComponentID]component.Extension {
	// TODO: A pipeline may want to have extensions for authentication;
	// this would not be hard to add here but it is not implemented in this PoC.
	return map[config.ComponentID]component.Extension{}
}

// GetExporters implements the component.Host interface.
func (h *host) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	// Return the only exporter available on the pipeline.
	return map[config.DataType]map[config.ComponentID]component.Exporter{
		h.datatype: {h.exporterID: h.exporter},
	}
}

// Pipeline is a telemetry pipeline.
type Pipeline struct {
	// components is the list of components in the pipeline:
	// - components[i+1] sends data to components[i]
	// - components[0] is the exporter
	// - components[-1] is the receiver.
	components []component.Component

	// host is the pipeline's component.Host
	host *host

	shutdownChan chan struct{}
}

// Run the pipeline until one component errors out.
func (p *Pipeline) Run(ctx context.Context) error {
	for _, component := range p.components {
		err := component.Start(ctx, p.host)
		if err != nil {
			// TODO: Components up to this point should be shutdown.
			return err
		}
	}

	for {
		select {
		case err := <-p.host.asyncErrorChannel:
			shutdownErr := p.shutdown(ctx)
			return multierr.Append(err, shutdownErr)
		case <-ctx.Done():
			return p.shutdown(ctx)
		case <-p.shutdownChan:
			return p.shutdown(ctx)
		}
	}

}

// shutdown the pipeline components.
func (p *Pipeline) shutdown(ctx context.Context) (err error) {
	for _, component := range p.components {
		err = multierr.Append(err, component.Shutdown(ctx))
	}
	return
}

// Shutdown the pipeline.
func (p *Pipeline) Shutdown() {
	close(p.shutdownChan)
}

// Builder builds a pipeline.
type Builder struct {
	set       component.TelemetrySettings
	buildInfo component.BuildInfo
	factories component.Factories
}

// NewBuilder creates a pipeline builder.
func NewBuilder(
	set component.TelemetrySettings,
	buildInfo component.BuildInfo,
	factories component.Factories,
) *Builder {

	// TODO: Should the Exporter|Receiver|ProcessorCreateSettings be passed instead?
	return &Builder{
		set:       set,
		buildInfo: buildInfo,
		factories: factories,
	}
}

// BuildMetricsPipeline creates a metrics pipeline based on the configuration of components.
// Components' configuration need to have a valid component ID.
//
// TODO: Some missing functionality here:
// - Factories can't be reused.
// - Extensions are not suppported.
// - Exactly one receiver and exporter is allowed.
func (b *Builder) BuildMetricsPipeline(
	ctx context.Context,
	receiver config.Receiver,
	processors []config.Processor,
	exporter config.Exporter,
) (*Pipeline, error) {

	// TODO: The logger is passed as-is; in a real implementation,
	// we would want to add fields for context as done in service.Service.
	pipeline := &Pipeline{
		shutdownChan: make(chan struct{}),
	}

	eFactory, ok := b.factories.Exporters[exporter.ID().Type()]
	if !ok {
		return nil, fmt.Errorf("factory not found for %q exporter", exporter.ID())
	}

	c, err := eFactory.CreateMetricsExporter(
		ctx,
		component.ExporterCreateSettings{
			BuildInfo:         b.buildInfo,
			TelemetrySettings: b.set,
		},
		exporter,
	)
	if err != nil {
		return nil, err
	}
	pipeline.components = append(pipeline.components, c)
	pipeline.host = &host{
		factories:         b.factories,
		datatype:          config.MetricsDataType,
		exporterID:        exporter.ID(),
		exporter:          c,
		asyncErrorChannel: make(chan error),
	}

	for i := len(processors) - 1; i >= 0; i-- {
		processor := processors[i]
		pFactory, procOk := b.factories.Processors[processor.ID().Type()]
		if !procOk {
			return nil, fmt.Errorf("factory not found for %q processor", processor.ID())
		}
		c, err = pFactory.CreateMetricsProcessor(
			ctx,
			component.ProcessorCreateSettings{
				BuildInfo:         b.buildInfo,
				TelemetrySettings: b.set,
			},
			processor, c)
		if err != nil {
			return nil, err
		}
		pipeline.components = append(pipeline.components, c)
	}

	rFactory, ok := b.factories.Receivers[receiver.ID().Type()]
	if !ok {
		return nil, fmt.Errorf("factory not found for %q receiver", receiver.ID())
	}
	r, err := rFactory.CreateMetricsReceiver(ctx,
		component.ReceiverCreateSettings{
			BuildInfo:         b.buildInfo,
			TelemetrySettings: b.set,
		},
		receiver, c,
	)
	if err != nil {
		return nil, err
	}
	pipeline.components = append(pipeline.components, r)

	return pipeline, nil
}
