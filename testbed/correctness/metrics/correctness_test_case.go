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

package metrics

import (
	"log"
	"testing"
	"time"

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
	collector testbed.OtelcolRunner
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
	tc.collector = testbed.NewInProcessCollector(componentFactories(tc.t))
	_, err := tc.collector.PrepareConfig(correctness.CreateConfigYaml(tc.sender, tc.receiver, nil, "metrics"))
	require.NoError(tc.t, err)
	rd, err := newResultsDir(tc.t.Name())
	require.NoError(tc.t, err)
	err = rd.mkDir()
	require.NoError(tc.t, err)
	fname, err := rd.fullPath("agent.log")
	require.NoError(tc.t, err)
	log.Println("starting collector")
	err = tc.collector.Start(testbed.StartParams{
		Name:        "Agent",
		LogFilePath: fname,
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
	for {
		select {
		case <-time.After(10 * time.Second):
			tc.t.Fatal("Deadline exceeded while waiting to receive metrics")
			return
		case <-tc.harness.allMetricsReceived:
			log.Println("all metrics received")
			return
		}
	}
}

func componentFactories(t *testing.T) component.Factories {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	return factories
}
