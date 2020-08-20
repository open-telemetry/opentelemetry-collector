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

package routingprocessor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var (
	errNoExporters            = errors.New("no exporters defined for the route")
	errNoTableItems           = errors.New("the routing table is empty")
	errNoMissingFromAttribute = errors.New("the FromAttribute property is empty")
	errExporterNotFound       = errors.New("exporter not found")
)

var _ component.TraceProcessor = (*processorImp)(nil)

type processorImp struct {
	logger *zap.Logger
	config Config

	defaultTraceExporters []component.TraceExporter
	traceExporters        map[string][]component.TraceExporter
}

// Crete new processor
func newProcessor(logger *zap.Logger, cfg configmodels.Exporter) (*processorImp, error) {
	logger.Info("building processor")

	oCfg := cfg.(*Config)

	// validate that every route has at least one exporter
	for _, item := range oCfg.Table {
		if len(item.Exporters) == 0 {
			return nil, fmt.Errorf("invalid route %s: %w", item.Value, errNoExporters)
		}
	}

	// validate that there's at least one item in the table
	if len(oCfg.Table) == 0 {
		return nil, fmt.Errorf("invalid routing table: %w", errNoTableItems)
	}

	// we also need a "FromAttribute" value
	if len(oCfg.FromAttribute) == 0 {
		return nil, fmt.Errorf("invalid attribute to read the route's value from: %w", errNoMissingFromAttribute)
	}

	return &processorImp{
		logger:         logger,
		config:         *oCfg,
		traceExporters: make(map[string][]component.TraceExporter),
	}, nil
}

func (e *processorImp) Start(_ context.Context, host component.Host) error {
	// first, let's build a map of exporter names with the exporter instances
	source := host.GetExporters()
	availableExporters := map[string]component.TraceExporter{}
	for k, exp := range source[configmodels.TracesDataType] {
		traceExp, ok := exp.(component.TraceExporter)
		if !ok {
			return fmt.Errorf("the exporter %q isn't a trace exporter", k.Name())
		}
		availableExporters[k.Name()] = traceExp
	}

	// default exporters
	if err := e.registerExportersForDefaultRoute(availableExporters, e.config.DefaultExporters); err != nil {
		return err
	}

	// exporters for each defined value
	for _, item := range e.config.Table {
		if err := e.registerExportersForRoute(item.Value, availableExporters, item.Exporters); err != nil {
			return err
		}
	}

	return nil
}

func (e *processorImp) registerExportersForDefaultRoute(available map[string]component.TraceExporter, requested []string) error {
	for _, exp := range requested {
		v, ok := available[exp]
		if !ok {
			return fmt.Errorf("error registering default exporter %q: %w", exp, errExporterNotFound)
		}
		e.defaultTraceExporters = append(e.defaultTraceExporters, v)
	}

	return nil
}

func (e *processorImp) registerExportersForRoute(route string, available map[string]component.TraceExporter, requested []string) error {
	for _, exp := range requested {
		v, ok := available[exp]
		if !ok {
			return fmt.Errorf("error registering route %q for exporter %q: %w", route, exp, errExporterNotFound)
		}
		e.traceExporters[route] = append(e.traceExporters[route], v)
	}

	return nil
}

func (e *processorImp) Shutdown(context.Context) error {
	return nil
}

func (e *processorImp) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	value := e.extractValueFromContext(ctx)
	if len(value) == 0 {
		// the attribute's value hasn't been found, send data to the default exporter
		return e.pushDataToExporters(ctx, td, e.defaultTraceExporters)
	}

	if _, ok := e.traceExporters[value]; !ok {
		// the value has been found, but there are no exporters for the value
		return e.pushDataToExporters(ctx, td, e.defaultTraceExporters)
	}

	// found the appropriate router, using it
	return e.pushDataToExporters(ctx, td, e.traceExporters[value])
}

func (e *processorImp) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (e *processorImp) pushDataToExporters(ctx context.Context, td pdata.Traces, exporters []component.TraceExporter) error {
	// TODO: determine the proper action when errors happen
	for _, exp := range exporters {
		if err := exp.ConsumeTraces(ctx, td); err != nil {
			return err
		}
	}

	return nil
}

func (e *processorImp) extractValueFromContext(ctx context.Context) string {
	// right now, we only support looking up attributes from requests that have gone through the gRPC server
	// in that case, it will add the HTTP headers as context metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// we have gRPC metadata in the context but does it have our key?
	values, ok := md[strings.ToLower(e.config.FromAttribute)]
	if !ok {
		return ""
	}

	if len(values) > 1 {
		e.logger.Debug("more than one value found for the attribute, using only the first", zap.Strings("values", values), zap.String("attribute", e.config.FromAttribute))
	}

	return values[0]
}
