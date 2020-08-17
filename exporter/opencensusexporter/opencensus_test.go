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

package opencensusexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestSendTraces(t *testing.T) {
	sink := &exportertest.SinkTraceExporter{}
	rFactory := opencensusreceiver.NewFactory()
	rCfg := rFactory.CreateDefaultConfig().(*opencensusreceiver.Config)
	endpoint := testutil.GetAvailableLocalAddress(t)
	rCfg.GRPCServerSettings.NetAddr.Endpoint = endpoint
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	recv, err := rFactory.CreateTraceReceiver(context.Background(), params, rCfg, sink)
	assert.NoError(t, err)
	assert.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, recv.Shutdown(context.Background()))
	})

	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: endpoint,
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	cfg.NumWorkers = 1
	exp, err := factory.CreateTraceExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	td := testdata.GenerateTraceDataOneSpan()
	octd := internaldata.TraceDataToOC(testdata.GenerateTraceDataOneSpan())
	require.Len(t, octd, 1)
	assert.NoError(t, exp.ConsumeTraceData(context.Background(), octd[0]))
	testutil.WaitFor(t, func() bool {
		return len(sink.AllTraces()) == 1
	})
	traces := sink.AllTraces()
	require.Len(t, traces, 1)
	assert.Equal(t, td, traces[0])

	// Sending data no Node.
	octd[0].Node = nil
	assert.NoError(t, exp.ConsumeTraceData(context.Background(), octd[0]))
	testutil.WaitFor(t, func() bool {
		return len(sink.AllTraces()) == 2
	})
	traces = sink.AllTraces()
	require.Len(t, traces, 2)
	assert.Equal(t, td, traces[1])
}

func TestSendTraces_NoBackend(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateTraceExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	td := internaldata.TraceDataToOC(testdata.GenerateTraceDataOneSpan())
	for i := 0; i < 10000; i++ {
		assert.Error(t, exp.ConsumeTraceData(context.Background(), td[0]))
	}
}

func TestSendTraces_AfterStop(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateTraceExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	assert.NoError(t, exp.Shutdown(context.Background()))

	td := internaldata.TraceDataToOC(testdata.GenerateTraceDataOneSpan())
	assert.Error(t, exp.ConsumeTraceData(context.Background(), td[0]))
}

func TestSendMetrics(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}
	rFactory := opencensusreceiver.NewFactory()
	rCfg := rFactory.CreateDefaultConfig().(*opencensusreceiver.Config)
	endpoint := testutil.GetAvailableLocalAddress(t)
	rCfg.GRPCServerSettings.NetAddr.Endpoint = endpoint
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	recv, err := rFactory.CreateMetricsReceiver(context.Background(), params, rCfg, sink)
	assert.NoError(t, err)
	assert.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, recv.Shutdown(context.Background()))
	})

	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: endpoint,
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	cfg.NumWorkers = 1
	exp, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	md := testdata.GenerateMetricDataOneMetric()
	ocmd := internaldata.MetricDataToOC(testdata.GenerateMetricDataOneMetric())
	require.Len(t, ocmd, 1)
	assert.NoError(t, exp.ConsumeMetricsData(context.Background(), ocmd[0]))
	testutil.WaitFor(t, func() bool {
		return len(sink.AllMetrics()) == 1
	})
	metrics := sink.AllMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, md, pdatautil.MetricsToInternalMetrics(metrics[0]))

	// Sending data no node.
	ocmd[0].Node = nil
	assert.NoError(t, exp.ConsumeMetricsData(context.Background(), ocmd[0]))
	testutil.WaitFor(t, func() bool {
		return len(sink.AllMetrics()) == 2
	})
	metrics = sink.AllMetrics()
	require.Len(t, metrics, 2)
	assert.Equal(t, md, pdatautil.MetricsToInternalMetrics(metrics[1]))
}

func TestSendMetrics_NoBackend(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	md := internaldata.MetricDataToOC(testdata.GenerateMetricDataOneMetric())
	for i := 0; i < 10000; i++ {
		assert.Error(t, exp.ConsumeMetricsData(context.Background(), md[0]))
	}
}

func TestSendMetrics_AfterStop(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	assert.NoError(t, exp.Shutdown(context.Background()))

	md := internaldata.MetricDataToOC(testdata.GenerateMetricDataOneMetric())
	assert.Error(t, exp.ConsumeMetricsData(context.Background(), md[0]))
}
