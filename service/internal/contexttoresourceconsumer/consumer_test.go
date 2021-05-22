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

package contexttoresourceconsumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/auth"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

var (
	raw    = "jdoe:password"
	sub    = "jdoe"
	groups = []string{"tenant1", "tenant2"}
)

func TestTraces(t *testing.T) {
	testCases := []struct {
		desc string
		td   pdata.Traces
	}{
		{
			desc: "one-span",
			td:   testdata.GenerateTracesOneSpan(),
		},
		{
			desc: "one-empty-resource-span",
			td:   testdata.GenerateTracesOneEmptyResourceSpans(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			next := new(consumertest.TracesSink)
			tc, err := NewTraces(next)
			require.NoError(t, err)

			ctx := prepareContext()

			// test
			tc.ConsumeTraces(ctx, tC.td)

			// verify
			for _, trace := range next.AllTraces() {
				assertValuesInAuth(t, trace.Auth(), raw, sub, groups)
			}

		})
	}
}

func TestMetrics(t *testing.T) {
	testCases := []struct {
		desc string
		td   pdata.Metrics
	}{
		{
			desc: "one-metric",
			td:   testdata.GenerateMetricsOneMetric(),
		},
		{
			desc: "no-resource",
			td:   testdata.GenerateMetricsOneMetricNoResource(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			next := new(consumertest.MetricsSink)
			tc, err := NewMetrics(next)
			require.NoError(t, err)

			ctx := prepareContext()

			// test
			tc.ConsumeMetrics(ctx, tC.td)

			// verify
			for _, metric := range next.AllMetrics() {
				assertValuesInAuth(t, metric.Auth(), raw, sub, groups)
			}
		})
	}
}

func TestLogs(t *testing.T) {
	testCases := []struct {
		desc string
		td   pdata.Logs
	}{
		{
			desc: "one-log",
			td:   testdata.GenerateLogsOneLogRecord(),
		},
		{
			desc: "no-logs",
			td:   testdata.GenerateLogsNoLogRecords(),
		},
		{
			desc: "no-resource",
			td:   testdata.GenerateLogsOneLogRecordNoResource(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			next := new(consumertest.LogsSink)
			tc, err := NewLogs(next)
			require.NoError(t, err)

			ctx := prepareContext()

			// test
			tc.ConsumeLogs(ctx, tC.td)

			// verify
			for _, log := range next.AllLogs() {
				assertValuesInAuth(t, log.Auth(), raw, sub, groups)
			}
		})
	}
}

func assertValuesInAuth(t *testing.T, auth *pdata.Auth, raw, sub string, groups []string) {
	assert.NotNil(t, auth)
	assert.Equal(t, raw, auth.Raw)
	assert.Equal(t, sub, auth.Subject)
	assert.Equal(t, groups, auth.Groups)
}

func prepareContext() context.Context {
	ctx := context.Background()
	ctx = auth.NewContextFromRaw(ctx, raw)
	ctx = auth.NewContextFromSubject(ctx, sub)
	ctx = auth.NewContextFromMemberships(ctx, groups)
	return ctx
}
