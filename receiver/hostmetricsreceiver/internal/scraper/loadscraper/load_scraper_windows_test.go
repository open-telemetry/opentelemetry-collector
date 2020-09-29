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

// +build windows

package loadscraper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
)

func TestStartSampling(t *testing.T) {
	// override sampling frequency to 2ms
	samplingFrequency = 2 * time.Millisecond

	// startSampling should set up perf counter and start sampling
	startSampling(context.Background(), zap.NewNop())
	assertSamplingUnderway(t)

	// override the processor queue length perf counter with a mock
	// that will ensure a positive value is returned
	assert.IsType(t, &perfcounters.PerfLibScraper{}, samplerInstance.perfCounterScraper)
	samplerInstance.perfCounterScraper = perfcounters.NewMockPerfCounterScraper(map[string]map[string][]int64{
		system: {processorQueueLength: {100}},
	})

	// second call to startSampling should succeed, but not do anything
	startSampling(context.Background(), zap.NewNop())
	assertSamplingUnderway(t)
	assert.IsType(t, &perfcounters.MockPerfCounterScraper{}, samplerInstance.perfCounterScraper)

	// ensure that a positive load avg is returned by a call to
	// "getSampledLoadAverages" which validates the value from the
	// mock perf counter was used
	require.Eventually(t, func() bool {
		avgLoadValues, err := getSampledLoadAverages()
		assert.NoError(t, err)
		return avgLoadValues.Load1 > 0 && avgLoadValues.Load5 > 0 && avgLoadValues.Load15 > 0
	}, time.Second, time.Millisecond, "Load Avg was not set after 1s")

	// sampling should continue after first call to stopSampling since
	// startSampling was called twice
	stopSampling(context.Background())
	assertSamplingUnderway(t)

	// second call to stopSampling should close perf counter, stop
	// sampling, and clean up the sampler
	stopSampling(context.Background())
	assertSamplingStopped(t)
}

func assertSamplingUnderway(t *testing.T) {
	assert.NotNil(t, samplerInstance)
	assert.NotNil(t, samplerInstance.perfCounterScraper)

	select {
	case <-samplerInstance.done:
		assert.Fail(t, "Load scraper sampling done channel unexpectedly closed")
	default:
	}
}

func assertSamplingStopped(t *testing.T) {
	select {
	case <-samplerInstance.done:
	default:
		assert.Fail(t, "Load scraper sampling done channel not closed")
	}
}

func TestSampleLoad(t *testing.T) {
	counterReturnValues := []int64{10, 20, 30, 40, 50}
	mockPerfCounterScraper := perfcounters.NewMockPerfCounterScraper(map[string]map[string][]int64{
		system: {processorQueueLength: counterReturnValues},
	})

	samplerInstance = &sampler{perfCounterScraper: mockPerfCounterScraper}

	for i := 0; i < len(counterReturnValues); i++ {
		samplerInstance.sampleLoad()
	}

	assert.Equal(t, calcExpectedLoad(counterReturnValues, loadAvgFactor1m), samplerInstance.loadAvg1m)
	assert.Equal(t, calcExpectedLoad(counterReturnValues, loadAvgFactor5m), samplerInstance.loadAvg5m)
	assert.Equal(t, calcExpectedLoad(counterReturnValues, loadAvgFactor15m), samplerInstance.loadAvg15m)
}

func calcExpectedLoad(scrapedValues []int64, loadAvgFactor float64) float64 {
	// replicate the calculations that should be performed to determine the exponentially
	// weighted moving averages based on the specified scraped values
	var expectedLoad float64
	for i := 0; i < len(scrapedValues); i++ {
		expectedLoad = expectedLoad*loadAvgFactor + float64(scrapedValues[i])*(1-loadAvgFactor)
	}
	return expectedLoad
}

func Benchmark_SampleLoad(b *testing.B) {
	s, _ := newSampler(zap.NewNop())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		s.sampleLoad()
	}
}
