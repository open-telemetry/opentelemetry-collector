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

func TestWrapResourceScraper(t *testing.T) {
	ts := &testResourceScraper{
		t:   t,
		err: nil,
	}
	obss := WrapResourceScraper(ts, "test")
	assert.NoError(t, obss.Initialize(context.Background()))
	rms, err := obss.ScrapeMetrics(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, generateResourceMetricsSlice(), rms)
	assert.NoError(t, obss.Close(context.Background()))
}

func TestWrapResourceScraper_Error(t *testing.T) {
	ts := &testResourceScraper{
		t:   t,
		err: fmt.Errorf("my error"),
	}
	obss := WrapResourceScraper(ts, "test")
	assert.Error(t, obss.Initialize(context.Background()))
	rms, err := obss.ScrapeMetrics(context.Background())
	assert.Error(t, err)
	assert.EqualValues(t, generateResourceMetricsSlice(), rms)
	assert.Error(t, obss.Close(context.Background()))
}

type testResourceScraper struct {
	t   *testing.T
	err error
}

func (s *testResourceScraper) Initialize(_ context.Context) error {
	return s.err
}

func (s *testResourceScraper) Close(_ context.Context) error {
	return s.err
}

// ScrapeMetrics
func (s *testResourceScraper) ScrapeMetrics(ctx context.Context) (pdata.ResourceMetricsSlice, error) {
	assert.NotNil(s.t, trace.FromContext(ctx))
	return generateResourceMetricsSlice(), s.err
}

func generateResourceMetricsSlice() pdata.ResourceMetricsSlice {
	return testdata.GenerateMetricsOneMetric().ResourceMetrics()
}
