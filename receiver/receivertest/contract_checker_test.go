// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
)

// This file is an example that demonstrates how to use the CheckConsumeContract() function.
// We declare a trivial example receiver, a data generator and then use them in TestConsumeContract().

type exampleReceiver struct {
	nextLogsConsumer    consumer.Logs
	nextTracesConsumer  consumer.Traces
	nextMetricsConsumer consumer.Metrics
}

func (s *exampleReceiver) Start(context.Context, component.Host) error {
	return nil
}

func (s *exampleReceiver) Shutdown(context.Context) error {
	return nil
}

func (s *exampleReceiver) ReceiveLogs(data plog.Logs) {
	// This very simple implementation demonstrates how a single items receiving should happen.
	for {
		err := s.nextLogsConsumer.ConsumeLogs(context.Background(), data)
		if err != nil {
			// The next consumer returned an error.
			if !consumererror.IsPermanent(err) {
				// It is not a permanent error, so we must retry sending it again. In network-based
				// receivers instead we can ask our sender to re-retry the same data again later.
				// We may also pause here a bit if we don't want to hammer the next consumer.
				continue
			}
		}
		// If we are hear either the ConsumeLogs returned success or it returned a permanent error.
		// In either case we don't need to retry the same data, we are done.
		return
	}
}

func (s *exampleReceiver) ReceiveMetrics(data pmetric.Metrics) {
	// This very simple implementation demonstrates how a single items receiving should happen.
	for {
		err := s.nextMetricsConsumer.ConsumeMetrics(context.Background(), data)
		if err != nil {
			// The next consumer returned an error.
			if !consumererror.IsPermanent(err) {
				// It is not a permanent error, so we must retry sending it again. In network-based
				// receivers instead we can ask our sender to re-retry the same data again later.
				// We may also pause here a bit if we don't want to hammer the next consumer.
				continue
			}
		}
		// If we are hear either the ConsumeLogs returned success or it returned a permanent error.
		// In either case we don't need to retry the same data, we are done.
		return
	}
}

func (s *exampleReceiver) ReceiveTraces(data ptrace.Traces) {
	// This very simple implementation demonstrates how a single items receiving should happen.
	for {
		err := s.nextTracesConsumer.ConsumeTraces(context.Background(), data)
		if err != nil {
			// The next consumer returned an error.
			if !consumererror.IsPermanent(err) {
				// It is not a permanent error, so we must retry sending it again. In network-based
				// receivers instead we can ask our sender to re-retry the same data again later.
				// We may also pause here a bit if we don't want to hammer the next consumer.
				continue
			}
		}
		// If we are hear either the ConsumeLogs returned success or it returned a permanent error.
		// In either case we don't need to retry the same data, we are done.
		return
	}
}

// A config for exampleReceiver.
type exampleReceiverConfig struct {
	generator Generator
}

// A generator that can send data to exampleReceiver.
type exampleLogGenerator struct {
	t           *testing.T
	receiver    *exampleReceiver
	sequenceNum int64
}

func (g *exampleLogGenerator) Start() {
	g.sequenceNum = 0
}

func (g *exampleLogGenerator) Stop() {}

func (g *exampleLogGenerator) Generate() []UniqueIDAttrVal {
	// Make sure the id is atomically incremented. Generate() may be called concurrently.
	id := UniqueIDAttrVal(strconv.FormatInt(atomic.AddInt64(&g.sequenceNum, 1), 10))

	data := CreateOneLogWithID(id)

	// Send the generated data to the receiver.
	g.receiver.ReceiveLogs(data)

	// And return the ids for bookkeeping by the test.
	return []UniqueIDAttrVal{id}
}

// A generator that can send data to exampleReceiver.
type exampleTraceGenerator struct {
	t           *testing.T
	receiver    *exampleReceiver
	sequenceNum int64
}

func (g *exampleTraceGenerator) Start() {
	g.sequenceNum = 0
}

func (g *exampleTraceGenerator) Stop() {}

func (g *exampleTraceGenerator) Generate() []UniqueIDAttrVal {
	// Make sure the id is atomically incremented. Generate() may be called concurrently.
	id := UniqueIDAttrVal(strconv.FormatInt(atomic.AddInt64(&g.sequenceNum, 1), 10))

	data := CreateOneSpanWithID(id)

	// Send the generated data to the receiver.
	g.receiver.ReceiveTraces(data)

	// And return the ids for bookkeeping by the test.
	return []UniqueIDAttrVal{id}
}

// A generator that can send data to exampleReceiver.
type exampleMetricGenerator struct {
	t           *testing.T
	receiver    *exampleReceiver
	sequenceNum int64
}

func (g *exampleMetricGenerator) Start() {
	g.sequenceNum = 0
}

func (g *exampleMetricGenerator) Stop() {}

func (g *exampleMetricGenerator) Generate() []UniqueIDAttrVal {
	// Make sure the id is atomically incremented. Generate() may be called concurrently.
	next := atomic.AddInt64(&g.sequenceNum, 1)
	id := UniqueIDAttrVal(strconv.FormatInt(next, 10))

	var data pmetric.Metrics
	switch next % 5 {
	case 0:
		data = CreateGaugeMetricWithID(id)
	case 1:
		data = CreateSumMetricWithID(id)
	case 2:
		data = CreateSummaryMetricWithID(id)
	case 3:
		data = CreateHistogramMetricWithID(id)
	case 4:
		data = CreateExponentialHistogramMetricWithID(id)
	}

	// Send the generated data to the receiver.
	g.receiver.ReceiveMetrics(data)

	// And return the ids for bookkeeping by the test.
	return []UniqueIDAttrVal{id}
}

func newExampleFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType("example_receiver"),
		func() component.Config {
			return &exampleReceiverConfig{}
		},
		receiver.WithLogs(createLog, component.StabilityLevelBeta),
		receiver.WithMetrics(createMetric, component.StabilityLevelBeta),
		receiver.WithTraces(createTrace, component.StabilityLevelBeta),
	)
}

func createTrace(_ context.Context, _ receiver.Settings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	rcv := &exampleReceiver{nextTracesConsumer: consumer}
	cfg.(*exampleReceiverConfig).generator.(*exampleTraceGenerator).receiver = rcv
	return rcv, nil
}

func createMetric(_ context.Context, _ receiver.Settings, cfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	rcv := &exampleReceiver{nextMetricsConsumer: consumer}
	cfg.(*exampleReceiverConfig).generator.(*exampleMetricGenerator).receiver = rcv
	return rcv, nil
}

func createLog(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	rcv := &exampleReceiver{nextLogsConsumer: consumer}
	cfg.(*exampleReceiverConfig).generator.(*exampleLogGenerator).receiver = rcv
	return rcv, nil
}

// TestConsumeContract is an example of testing of the receiver for the contract between the
// receiver and next consumer.
func TestConsumeContract(t *testing.T) {
	// Number of log records to send per scenario.
	const logsPerTest = 100

	generator := &exampleLogGenerator{t: t}
	cfg := &exampleReceiverConfig{generator: generator}

	params := CheckConsumeContractParams{
		T:             t,
		Factory:       newExampleFactory(),
		Signal:        pipeline.SignalLogs,
		Config:        cfg,
		Generator:     generator,
		GenerateCount: logsPerTest,
	}

	// Run the contract checker. This will trigger test failures if any problems are found.
	CheckConsumeContract(params)
}

// TestConsumeMetricsContract is an example of testing of the receiver for the contract between the
// receiver and next consumer.
func TestConsumeMetricsContract(t *testing.T) {
	// Number of metric data points to send per scenario.
	const metricsPerTest = 100

	generator := &exampleMetricGenerator{t: t}
	cfg := &exampleReceiverConfig{generator: generator}

	params := CheckConsumeContractParams{
		T:             t,
		Factory:       newExampleFactory(),
		Signal:        pipeline.SignalMetrics,
		Config:        cfg,
		Generator:     generator,
		GenerateCount: metricsPerTest,
	}

	// Run the contract checker. This will trigger test failures if any problems are found.
	CheckConsumeContract(params)
}

// TestConsumeTracesContract is an example of testing of the receiver for the contract between the
// receiver and next consumer.
func TestConsumeTracesContract(t *testing.T) {
	// Number of trace spans to send per scenario.
	const spansPerTest = 100

	generator := &exampleTraceGenerator{t: t}
	cfg := &exampleReceiverConfig{generator: generator}

	params := CheckConsumeContractParams{
		T:             t,
		Factory:       newExampleFactory(),
		Signal:        pipeline.SignalTraces,
		Config:        cfg,
		Generator:     generator,
		GenerateCount: spansPerTest,
	}

	// Run the contract checker. This will trigger test failures if any problems are found.
	CheckConsumeContract(params)
}

func TestIDSetFromDataPoint(t *testing.T) {
	require.Error(t, idSetFromDataPoint(map[UniqueIDAttrVal]bool{}, pcommon.NewMap()))
	m := pcommon.NewMap()
	m.PutStr("foo", "bar")
	require.Error(t, idSetFromDataPoint(map[UniqueIDAttrVal]bool{}, m))
	m.PutInt(UniqueIDAttrName, 64)
	require.Error(t, idSetFromDataPoint(map[UniqueIDAttrVal]bool{}, m))
	m.PutStr(UniqueIDAttrName, "myid")
	result := map[UniqueIDAttrVal]bool{}
	require.NoError(t, idSetFromDataPoint(result, m))
	require.True(t, result["myid"])
}

func TestBadMetricPoint(t *testing.T) {
	for _, test := range []struct {
		name    string
		metrics pmetric.Metrics
	}{
		{
			name: "gauge",
			metrics: func() pmetric.Metrics {
				m := pmetric.NewMetrics()
				m.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()
				return m
			}(),
		},
		{
			name: "sum",
			metrics: func() pmetric.Metrics {
				m := pmetric.NewMetrics()
				m.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum().DataPoints().AppendEmpty()
				return m
			}(),
		},
		{
			name: "summary",
			metrics: func() pmetric.Metrics {
				m := pmetric.NewMetrics()
				m.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySummary().DataPoints().AppendEmpty()
				return m
			}(),
		},
		{
			name: "histogram",
			metrics: func() pmetric.Metrics {
				m := pmetric.NewMetrics()
				m.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram().DataPoints().AppendEmpty()
				return m
			}(),
		},
		{
			name: "exponential histogram",
			metrics: func() pmetric.Metrics {
				m := pmetric.NewMetrics()
				m.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
				return m
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := idSetFromMetrics(test.metrics)
			require.Error(t, err)
		})
	}
}
