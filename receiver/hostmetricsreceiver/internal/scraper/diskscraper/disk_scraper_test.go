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

package diskscraper

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrapeMetrics(t *testing.T) {
	scraper := newDiskScraper(context.Background(), &Config{})

	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize disk scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assert.GreaterOrEqual(t, metrics.Len(), 2)

	assertDiskMetricValid(t, metrics.At(0), diskIODescriptor, 0)
	assertDiskMetricValid(t, metrics.At(1), diskOpsDescriptor, 0)

	if runtime.GOOS != "windows" {
		assertDiskMetricValid(t, metrics.At(2), diskTimeDescriptor, 0)
	}

	if runtime.GOOS == "linux" {
		assertDiskMetricValid(t, metrics.At(3), diskMergedDescriptor, 0)
	}
}

func assertDiskMetricValid(t *testing.T, metric pdata.Metric, expectedDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric.MetricDescriptor())
	if startTime != 0 {
		internal.AssertInt64MetricStartTimeEquals(t, metric, startTime)
	}
	assert.GreaterOrEqual(t, metric.Int64DataPoints().Len(), 2)
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, writeDirectionLabelValue)
}
