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
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/testutil"
)

const testPrefix = "a_test"

func TestOpenCensus(t *testing.T) {
	flagSet := new(flag.FlagSet)
	configtelemetry.Flags(flagSet)
	telemetry.Flags(flagSet)
	metricsPort := testutil.GetAvailablePort(t)
	require.NoError(t, flagSet.Parse([]string{
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
	}))

	otelColTelemetry := &colTelemetry{}
	asyncErrorChannel := make(chan error)
	require.NoError(t, otelColTelemetry.init(asyncErrorChannel, 0, zap.NewNop()))

	// All labels added to all collector metrics by default are listed below.
	// These labels are hard coded here in order to avoid inadvertent changes:
	// at this point changing labels should be treated as a breaking changing
	// and requires a good justification. The reason is that changes to metric
	// names or labels can break alerting, dashboards, etc that are used to
	// monitor the Collector in production deployments.
	mandatoryLabels := []string{
		"service_instance_id",
	}
	assertMetrics(t, testPrefix, metricsPort, mandatoryLabels)

	require.NoError(t, otelColTelemetry.shutdown())
}

func TestOpenCensus_StartStop(t *testing.T) {
	flagSet := new(flag.FlagSet)
	configtelemetry.Flags(flagSet)
	telemetry.Flags(flagSet)
	metricsPort := testutil.GetAvailablePort(t)
	require.NoError(t, flagSet.Parse([]string{
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
	}))

	otelColTelemetry := &colTelemetry{}
	asyncErrorChannel := make(chan error)
	require.NoError(t, otelColTelemetry.init(asyncErrorChannel, 0, zap.NewNop()))
	require.NoError(t, otelColTelemetry.shutdown())
	require.NoError(t, otelColTelemetry.init(asyncErrorChannel, 0, zap.NewNop()))
	require.NoError(t, otelColTelemetry.shutdown())
}

func assertMetrics(t *testing.T, prefix string, metricsPort uint16, mandatoryLabels []string) {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	require.NoError(t, err)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(bufio.NewReader(resp.Body))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.NotZero(t, len(parsed))

	for metricName, metricFamily := range parsed {
		t.Log(metricName)
		// require is used here so test fails with a single message.
		require.Truef(
			t,
			strings.HasPrefix(metricName, prefix),
			"expected prefix %q but string starts with %q", prefix, metricName[:len(prefix)+1]+"...")

		for _, metric := range metricFamily.Metric {
			var labelNames []string
			for _, labelPair := range metric.Label {
				labelNames = append(labelNames, *labelPair.Name)
			}

			for _, mandatoryLabel := range mandatoryLabels {
				// require is used here so test fails with a single message.
				require.Contains(t, labelNames, mandatoryLabel, "mandatory label %q not present", mandatoryLabel)
			}
		}
	}
}
