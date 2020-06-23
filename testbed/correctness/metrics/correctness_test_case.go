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

type CorrectnessTestCase struct {
	t         *testing.T
	sender    testbed.DataSender
	receiver  testbed.DataReceiver
	harness   *TestHarness
	collector *testbed.InProcessCollector
}

func NewCorrectnessTestCase(t *testing.T, sender testbed.DataSender, receiver testbed.DataReceiver, harness *TestHarness) *CorrectnessTestCase {
	return &CorrectnessTestCase{t: t, sender: sender, receiver: receiver, harness: harness}
}

func (tc *CorrectnessTestCase) StartCollector() {
	tc.collector = testbed.NewInProcessCollector(factories(tc.t), tc.sender.GetCollectorPort())
	_, err := tc.collector.PrepareConfig(correctness.CreateConfigYaml(tc.sender, tc.receiver, nil, "metrics"))
	require.NoError(tc.t, err)
	rd, err := testbed.NewResultsDir(tc.t.Name())
	require.NoError(tc.t, err)
	err = rd.MkDir()
	require.NoError(tc.t, err)
	fname, err := rd.FileName("agent.log")
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

func (tc *CorrectnessTestCase) StopCollector() {
	_, err := tc.collector.Stop()
	require.NoError(tc.t, err)
}

func (tc *CorrectnessTestCase) StartTestbedSender() {
	log.Println("starting testbed sender")
	err := tc.sender.Start()
	require.NoError(tc.t, err)
}

func (tc *CorrectnessTestCase) StartTestbedReceiver() {
	log.Println("starting testbed receiver")
	err := tc.receiver.Start(&testbed.MockTraceConsumer{}, tc.harness)
	require.NoError(tc.t, err)
}

func (tc *CorrectnessTestCase) StopTestbedReceiver() {
	log.Println("stopping testbed receiver")
	err := tc.receiver.Stop()
	require.NoError(tc.t, err)
}

func (tc *CorrectnessTestCase) SendFirstMetric() {
	tc.harness.sendNextMetric()
}

func (tc *CorrectnessTestCase) WaitForAllMetrics() {
	log.Println("selecting")
	select {
	case <-tc.harness.allMetricsReceived:
		log.Println("testHarness.allMetricsReceived")
	}
}

func factories(t *testing.T) component.Factories {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	return factories
}
