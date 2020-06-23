package metrics

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/defaultcomponents"
	"go.opentelemetry.io/collector/testbed/correctness"
	"go.opentelemetry.io/collector/testbed/testbed"
)

type correctnessTestCase struct {
	t         *testing.T
	sender    testbed.DataSender
	receiver  testbed.DataReceiver
	harness   *testHarness
	collector *testbed.InProcessCollector
}

func newCorrectnessTestCase(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	harness *testHarness,
) *correctnessTestCase {
	return &correctnessTestCase{t: t, sender: sender, receiver: receiver, harness: harness}
}

func (tc *correctnessTestCase) startCollector() {
	tc.collector = testbed.NewInProcessCollector(componentFactories(tc.t), tc.sender.GetCollectorPort())
	_, err := tc.collector.PrepareConfig(correctness.CreateConfigYaml(tc.sender, tc.receiver, nil, "metrics"))
	require.NoError(tc.t, err)
	rd, err := newResultsDir(tc.t.Name())
	require.NoError(tc.t, err)
	err = rd.mkDir()
	require.NoError(tc.t, err)
	fname, err := rd.fullPath("agent.log")
	require.NoError(tc.t, err)
	log.Println("starting collector")
	_, err = tc.collector.Start(testbed.StartParams{
		Name:        "Agent",
		LogFilePath: fname,
		Cmd:         "foo",
		CmdArgs:     []string{"--metrics-level=NONE"},
	})
	require.NoError(tc.t, err)
}

func (tc *correctnessTestCase) stopCollector() {
	_, err := tc.collector.Stop()
	require.NoError(tc.t, err)
}

func (tc *correctnessTestCase) startTestbedSender() {
	log.Println("starting testbed sender")
	err := tc.sender.Start()
	require.NoError(tc.t, err)
}

func (tc *correctnessTestCase) startTestbedReceiver() {
	log.Println("starting testbed receiver")
	err := tc.receiver.Start(&testbed.MockTraceConsumer{}, tc.harness, &testbed.MockLogConsumer{})
	require.NoError(tc.t, err)
}

func (tc *correctnessTestCase) stopTestbedReceiver() {
	log.Println("stopping testbed receiver")
	err := tc.receiver.Stop()
	require.NoError(tc.t, err)
}

func (tc *correctnessTestCase) sendFirstMetric() {
	tc.harness.sendNextMetric()
}

func (tc *correctnessTestCase) waitForAllMetrics() {
	log.Println("waiting for allMetricsReceived")
	<-tc.harness.allMetricsReceived
	log.Println("all metrics received")
}

func componentFactories(t *testing.T) component.Factories {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	return factories
}
