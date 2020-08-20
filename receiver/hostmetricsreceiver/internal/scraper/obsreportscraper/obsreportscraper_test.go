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

package obsreportscraper

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestWrapScraper(t *testing.T) {
	ts := &testScraper{
		t:   t,
		err: nil,
	}
	obss := WrapScraper(ts, "test")
	assert.NoError(t, obss.Initialize(context.Background()))
	ms, err := obss.ScrapeMetrics(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, generateMetricsSlice(), ms)
	assert.NoError(t, obss.Close(context.Background()))
}

func TestWrapScraper_Error(t *testing.T) {
	ts := &testScraper{
		t:   t,
		err: fmt.Errorf("my error"),
	}
	obss := WrapScraper(ts, "test")
	assert.Error(t, obss.Initialize(context.Background()))
	ms, err := obss.ScrapeMetrics(context.Background())
	assert.Error(t, err)
	assert.EqualValues(t, generateMetricsSlice(), ms)
	assert.Error(t, obss.Close(context.Background()))
}

type testScraper struct {
	t   *testing.T
	err error
}

func (s *testScraper) Initialize(_ context.Context) error {
	return s.err
}

func (s *testScraper) Close(_ context.Context) error {
	return s.err
}

// ScrapeMetrics
func (s *testScraper) ScrapeMetrics(ctx context.Context) (pdata.MetricSlice, error) {
	assert.NotNil(s.t, trace.FromContext(ctx))
	return generateMetricsSlice(), s.err
}

func generateMetricsSlice() pdata.MetricSlice {
	return testdata.GenerateMetricDataOneMetric().ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
}
