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

package processesscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/load"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processesscraper/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

var (
	expectProcessesCountMetric   = runtime.GOOS == "linux" || runtime.GOOS == "openbsd" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" || runtime.GOOS == "solaris"
	expectProcessesCreatedMetric = runtime.GOOS == "linux" || runtime.GOOS == "openbsd"
)

const startTime = 100 * 1e9

func TestScrape(t *testing.T) {
	type testCase struct {
		name         string
		getMiscStats func() (*load.MiscStat, error)
		getProcesses func() ([]proc, error)
		expectedErr  string
		validate     func(*testing.T, pdata.MetricSlice)
	}

	testCases := []testCase{{
		name:     "Standard",
		validate: validateRealData,
	}, {
		name:         "FakeData",
		getMiscStats: func() (*load.MiscStat, error) { return &fakeData, nil },
		getProcesses: func() ([]proc, error) { return fakeProcessesData, nil },
		validate:     validateFakeData,
	}, {
		name:         "ErrorFromMiscStat",
		getMiscStats: func() (*load.MiscStat, error) { return &load.MiscStat{}, errors.New("err1") },
		expectedErr:  "err1",
	}, {
		name:         "ErrorFromProcesses",
		getProcesses: func() ([]proc, error) { return nil, errors.New("err2") },
		expectedErr:  "err2",
	}, {
		name:         "ErrorFromProcessShouldBeIgnored",
		getProcesses: func() ([]proc, error) { return []proc{errProcess{}}, nil },
	}, {
		name:     "Validate Start Time",
		validate: validateStartTime,
	}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			scraper := newProcessesScraper(context.Background(), &Config{})
			err := scraper.start(context.Background(), componenttest.NewNopHost())
			assert.NoError(err, "Failed to initialize processes scraper: %v", err)

			// Override scraper methods if we are mocking out for this test case
			if test.getMiscStats != nil {
				scraper.getMiscStats = test.getMiscStats
			}
			if test.getProcesses != nil {
				scraper.getProcesses = test.getProcesses
			}
			scraper.startTime = startTime

			metrics, err := scraper.scrape(context.Background())

			expectedMetricCount := 0
			if expectProcessesCountMetric {
				expectedMetricCount++
			}
			if expectProcessesCreatedMetric {
				expectedMetricCount++
			}

			if (expectProcessesCountMetric || expectProcessesCreatedMetric) && test.expectedErr != "" {
				assert.EqualError(err, test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.Truef(isPartial, "expected partial scrape error, have %+v", err)
				if isPartial {
					assert.Equal(expectedMetricCount, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}

			if test.expectedErr == "" {
				assert.NoErrorf(err, "Failed to scrape metrics: %v", err)
			}

			assert.Equal(expectedMetricCount, metrics.Len())

			if test.validate != nil {
				test.validate(t, metrics)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func validateRealData(t *testing.T, metrics pdata.MetricSlice) {
	assert := assert.New(t)

	metricIndex := 0
	if expectProcessesCountMetric {
		countMetric := metrics.At(metricIndex)
		metricIndex++
		internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCount.New(), countMetric)

		assertContainsStatus := func(statusVal string) {
			for i := 0; i < countMetric.Sum().DataPoints().Len(); i++ {
				v, ok := countMetric.Sum().DataPoints().At(i).Attributes().Get(metadata.Labels.Status)
				if ok && v.StringVal() == statusVal {
					return
				}
			}
			assert.Failf("missing-metric", "metric is missing %q status label", statusVal)
		}
		assertContainsStatus(metadata.LabelStatus.Running)
		assertContainsStatus(metadata.LabelStatus.Blocked)
	}

	if expectProcessesCreatedMetric {
		createdMetric := metrics.At(metricIndex)
		internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCreated.New(), createdMetric)
		createdMetric = metrics.At(1)
		internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCreated.New(), createdMetric)
		assert.Equal(1, createdMetric.Sum().DataPoints().Len())
		assert.Equal(0, createdMetric.Sum().DataPoints().At(0).Attributes().Len())
	}
}

func validateStartTime(t *testing.T, metrics pdata.MetricSlice) {
	internal.AssertSumMetricStartTimeEquals(t, metrics.At(0), startTime)
	internal.AssertSumMetricStartTimeEquals(t, metrics.At(1), startTime)
}

var fakeData = load.MiscStat{
	ProcsCreated: 1,
	ProcsRunning: 2,
	ProcsBlocked: 3,
	ProcsTotal:   30,
}

var fakeProcessesData = []proc{
	fakeProcess("W"),
	fakeProcess("D"), fakeProcess("D"),
	fakeProcess("R"), fakeProcess("R"), fakeProcess("R"),
	fakeProcess("S"), fakeProcess("S"), fakeProcess("S"), fakeProcess("S"),
	fakeProcess("T"), fakeProcess("T"), fakeProcess("T"), fakeProcess("T"), fakeProcess("T"),
	fakeProcess("Z"), fakeProcess("Z"), fakeProcess("Z"), fakeProcess("Z"), fakeProcess("Z"), fakeProcess("Z"),
}

type errProcess struct{}

func (e errProcess) Status() (string, error) {
	return "", errors.New("errProcess")
}

type fakeProcess string

func (f fakeProcess) Status() (string, error) {
	return string(f), nil
}

func validateFakeData(t *testing.T, metrics pdata.MetricSlice) {
	assert := assert.New(t)
	metricIndex := 0
	if expectProcessesCountMetric {
		countMetric := metrics.At(metricIndex)
		metricIndex++
		internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCount.New(), countMetric)
		assert.Equal(7, countMetric.Sum().DataPoints().Len())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 0, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Blocked))
		assert.Equal(int64(3), countMetric.Sum().DataPoints().At(0).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 1, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Paging))
		assert.Equal(int64(1), countMetric.Sum().DataPoints().At(1).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 2, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Running))
		assert.Equal(int64(2), countMetric.Sum().DataPoints().At(2).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 3, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Sleeping))
		assert.Equal(int64(4), countMetric.Sum().DataPoints().At(3).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 4, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Stopped))
		assert.Equal(int64(5), countMetric.Sum().DataPoints().At(4).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 5, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Unknown))
		assert.Equal(int64(9), countMetric.Sum().DataPoints().At(5).IntVal())

		internal.AssertSumMetricHasAttributeValue(t, countMetric, 6, metadata.Labels.Status, pdata.NewAttributeValueString(metadata.LabelStatus.Zombies))
		assert.Equal(int64(6), countMetric.Sum().DataPoints().At(6).IntVal())
	}

	if expectProcessesCreatedMetric {
		createdMetric := metrics.At(metricIndex)
		internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCreated.New(), createdMetric)
		assert.Equal(1, createdMetric.Sum().DataPoints().Len())
		assert.Equal(0, createdMetric.Sum().DataPoints().At(0).Attributes().Len())
	}
}
