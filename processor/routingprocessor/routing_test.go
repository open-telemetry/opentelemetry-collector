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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	v1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
)

func TestRouteIsFoundForGRPCContexts(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}
	wg.Add(1)

	exp := &processorImp{
		config: Config{
			FromAttribute: "X-Tenant",
		},
		logger: zap.NewNop(),
		traceExporters: map[string][]component.TraceExporter{
			"acme": {
				&mockExporter{
					ConsumeTracesFunc: func(context.Context, pdata.Traces) error {
						wg.Done()
						return nil
					},
				},
			},
		},
	}
	traces := pdata.NewTraces()

	// test
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Tenant", "acme"))
	err := exp.ConsumeTraces(ctx, traces)

	// verify
	wg.Wait() // ensure that the exporter has been called
	assert.NoError(t, err)
}

func TestDefaultRouteIsUsedWhenRouteCantBeDetermined(t *testing.T) {
	for _, tt := range []struct {
		name string
		ctx  context.Context
	}{
		{
			"no key",
			context.Background(),
		},
		{
			"no value",
			metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Tenant", "acme")),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			wg := &sync.WaitGroup{}
			wg.Add(1)

			exp := &processorImp{
				config: Config{
					FromAttribute: "X-Tenant",
				},
				logger: zap.NewNop(),
				defaultTraceExporters: []component.TraceExporter{
					&mockExporter{
						ConsumeTracesFunc: func(context.Context, pdata.Traces) error {
							wg.Done()
							return nil
						},
					},
				},
			}
			traces := pdata.NewTraces()

			// test
			err := exp.ConsumeTraces(tt.ctx, traces)

			// verify
			wg.Wait() // ensure that the exporter has been called
			assert.NoError(t, err)

		})
	}
}

func TestRegisterExportersForValidRoute(t *testing.T) {
	//  prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)

	otlpExpFactory := otlpexporter.NewFactory()
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	otlpConfig := &otlpexporter.Config{
		ExporterSettings: configmodels.ExporterSettings{
			NameVal: "otlp",
			TypeVal: "otlp",
		},
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: "example.com:1234",
		},
	}
	otlpExp, err := otlpExpFactory.CreateTraceExporter(context.Background(), creationParams, otlpConfig)
	require.NoError(t, err)
	host := &mockHost{
		GetExportersFunc: func() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
			return map[configmodels.DataType]map[configmodels.Exporter]component.Exporter{
				configmodels.TracesDataType: {
					otlpConfig: otlpExp,
				},
			}
		},
	}

	// test
	exp.Start(context.Background(), host)

	// verify
	assert.Contains(t, exp.traceExporters["acme"], otlpExp)
}

func TestErrorRequestedExporterNotFoundForRoute(t *testing.T) {
	//  prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		FromAttribute: "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"non-existing"},
			},
		},
	})
	require.NoError(t, err)
	host := &mockHost{}

	// test
	err = exp.Start(context.Background(), host)

	// verify
	assert.True(t, errors.Is(err, errExporterNotFound))
}

func TestErrorRequestedExporterNotFoundForDefaultRoute(t *testing.T) {
	//  prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"non-existing"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)

	otlpExpFactory := otlpexporter.NewFactory()
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	otlpConfig := &otlpexporter.Config{
		ExporterSettings: configmodels.ExporterSettings{
			NameVal: "otlp",
			TypeVal: "otlp",
		},
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: "example.com:1234",
		},
	}
	otlpExp, err := otlpExpFactory.CreateTraceExporter(context.Background(), creationParams, otlpConfig)
	require.NoError(t, err)
	host := &mockHost{
		GetExportersFunc: func() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
			return map[configmodels.DataType]map[configmodels.Exporter]component.Exporter{
				configmodels.TracesDataType: {
					otlpConfig: otlpExp,
				},
			}
		},
	}

	// test
	err = exp.Start(context.Background(), host)

	// verify
	assert.True(t, errors.Is(err, errExporterNotFound))
}

func TestInvalidExporter(t *testing.T) {
	//  prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)

	otlpConfig := &otlpexporter.Config{
		ExporterSettings: configmodels.ExporterSettings{
			NameVal: "otlp",
			TypeVal: "otlp",
		},
	}
	host := &mockHost{
		GetExportersFunc: func() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
			return map[configmodels.DataType]map[configmodels.Exporter]component.Exporter{
				configmodels.TracesDataType: {
					otlpConfig: &mockComponent{},
				},
			}
		},
	}

	// test
	err = exp.Start(context.Background(), host)

	// verify
	assert.Error(t, err)
}

func TestValueFromExistingGRPCAttribute(t *testing.T) {
	// prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Tenant", "acme"))

	// test
	val := exp.extractValueFromContext(ctx)

	// verify
	assert.Equal(t, "acme", val)
}

func TestMultipleValuesFromExistingGRPCAttribute(t *testing.T) {
	// prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Tenant", "globex", "X-Tenant", "acme"))

	// test
	val := exp.extractValueFromContext(ctx)

	// verify
	assert.Equal(t, "globex", val)
}

func TestNoValuesFromExistingGRPCAttribute(t *testing.T) {
	// prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Tenant", ""))

	// test
	val := exp.extractValueFromContext(ctx)

	// verify
	assert.Equal(t, "", val)
}

func TestAttributeFromExistingGRPCContext(t *testing.T) {
	// prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{}))

	// test
	val := exp.extractValueFromContext(ctx)

	// verify
	assert.Equal(t, "", val)
}

func TestNoAttributeInContext(t *testing.T) {
	// prepare
	exp, err := newProcessor(zap.NewNop(), &Config{
		DefaultExporters: []string{"otlp"},
		FromAttribute:    "X-Tenant",
		Table: []RoutingTableItem{
			{
				Value:     "acme",
				Exporters: []string{"otlp"},
			},
		},
	})
	require.NoError(t, err)

	// test
	val := exp.extractValueFromContext(context.Background())

	// verify
	assert.Equal(t, "", val)
}

func TestFailedToPushDataToExporter(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}
	expectedErr := errors.New("some error")
	wg.Add(2)
	exp := &processorImp{
		logger: zap.NewNop(),
		traceExporters: map[string][]component.TraceExporter{
			"acme": {
				&mockExporter{
					ConsumeTracesFunc: func(context.Context, pdata.Traces) error {
						wg.Done()
						return nil
					},
				},
				&mockExporter{ // this is a cross-test with the scenario with multiple exporters
					ConsumeTracesFunc: func(context.Context, pdata.Traces) error {
						wg.Done()
						return expectedErr
					},
				},
			},
		},
	}
	traces := pdata.TracesFromOtlp([]*v1.ResourceSpans{{
		InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{{
			Spans: []*v1.Span{{}},
		}},
	}})

	// test
	err := exp.pushDataToExporters(context.Background(), traces, exp.traceExporters["acme"])

	// verify
	wg.Wait() // ensure that the exporter has been called
	assert.Equal(t, expectedErr, err)
}

func TestProcessorCapabilities(t *testing.T) {
	// prepare
	config := &Config{
		FromAttribute: "X-Tenant",
		Table: []RoutingTableItem{{
			Exporters: []string{"otlp"},
		}},
	}

	// test
	p, err := newProcessor(zap.NewNop(), config)
	caps := p.GetCapabilities()

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, true, caps.MutatesConsumedData)
}

type mockHost struct {
	componenttest.NopHost
	GetExportersFunc func() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter
}

func (m *mockHost) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	if m.GetExportersFunc != nil {
		return m.GetExportersFunc()
	}
	return m.NopHost.GetExporters()
}

type mockComponent struct{}

func (m *mockComponent) Start(context.Context, component.Host) error {
	return nil
}
func (m *mockComponent) Shutdown(context.Context) error {
	return nil
}

type mockExporter struct {
	mockComponent
	ConsumeTracesFunc func(ctx context.Context, td pdata.Traces) error
}

func (m *mockExporter) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if m.ConsumeTracesFunc != nil {
		return m.ConsumeTracesFunc(ctx, td)
	}
	return nil
}
