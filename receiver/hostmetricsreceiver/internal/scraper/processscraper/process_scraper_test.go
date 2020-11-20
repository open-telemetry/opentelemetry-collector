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

package processscraper

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/translator/conventions"
)

func skipTestOnUnsupportedOS(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skipf("skipping test on %v", runtime.GOOS)
	}
}

func TestScrape(t *testing.T) {
	skipTestOnUnsupportedOS(t)

	const bootTime = 100
	const expectedStartTime = 100 * 1e9

	scraper, err := newProcessScraper(&Config{})
	scraper.bootTime = func() (uint64, error) { return bootTime, nil }
	require.NoError(t, err, "Failed to create process scraper: %v", err)
	err = scraper.start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to initialize process scraper: %v", err)

	resourceMetrics, err := scraper.scrape(context.Background())

	// may receive some partial errors as a result of attempting to:
	// a) read native system processes on Windows (e.g. Registry process)
	// b) read info on processes that have just terminated
	//
	// so validate that we have less processes that were failed to be scraped
	// than processes that were successfully scraped & some valid data is
	// returned
	if err != nil {
		require.True(t, consumererror.IsPartialScrapeError(err))
		noProcessesScraped := resourceMetrics.Len()
		noProcessesErrored := err.(consumererror.PartialScrapeError).Failed
		require.Lessf(t, noProcessesErrored, noProcessesScraped, "Failed to scrape metrics - more processes failed to be scraped than were successfully scraped: %v", err)
	}

	require.Greater(t, resourceMetrics.Len(), 1)
	assertProcessResourceAttributesExist(t, resourceMetrics)
	assertCPUTimeMetricValid(t, resourceMetrics, expectedStartTime)
	assertMemoryUsageMetricValid(t, physicalMemoryUsageDescriptor, resourceMetrics)
	assertMemoryUsageMetricValid(t, virtualMemoryUsageDescriptor, resourceMetrics)
	assertDiskIOMetricValid(t, resourceMetrics, expectedStartTime)
	assertSameTimeStampForAllMetricsWithinResource(t, resourceMetrics)
}

func assertProcessResourceAttributesExist(t *testing.T, resourceMetrics pdata.ResourceMetricsSlice) {
	for i := 0; i < resourceMetrics.Len(); i++ {
		attr := resourceMetrics.At(0).Resource().Attributes()
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessID)
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessExecutableName)
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessExecutablePath)
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessCommand)
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessCommandLine)
		internal.AssertContainsAttribute(t, attr, conventions.AttributeProcessOwner)
	}
}

func assertCPUTimeMetricValid(t *testing.T, resourceMetrics pdata.ResourceMetricsSlice, startTime pdata.TimestampUnixNano) {
	cpuTimeMetric := getMetric(t, cpuTimeDescriptor, resourceMetrics)
	internal.AssertDescriptorEqual(t, cpuTimeDescriptor, cpuTimeMetric)
	if startTime != 0 {
		internal.AssertDoubleSumMetricStartTimeEquals(t, cpuTimeMetric, startTime)
	}
	internal.AssertDoubleSumMetricLabelHasValue(t, cpuTimeMetric, 0, stateLabelName, userStateLabelValue)
	internal.AssertDoubleSumMetricLabelHasValue(t, cpuTimeMetric, 1, stateLabelName, systemStateLabelValue)
	if runtime.GOOS == "linux" {
		internal.AssertDoubleSumMetricLabelHasValue(t, cpuTimeMetric, 2, stateLabelName, waitStateLabelValue)
	}
}

func assertMemoryUsageMetricValid(t *testing.T, descriptor pdata.Metric, resourceMetrics pdata.ResourceMetricsSlice) {
	memoryUsageMetric := getMetric(t, descriptor, resourceMetrics)
	internal.AssertDescriptorEqual(t, descriptor, memoryUsageMetric)
}

func assertDiskIOMetricValid(t *testing.T, resourceMetrics pdata.ResourceMetricsSlice, startTime pdata.TimestampUnixNano) {
	diskIOMetric := getMetric(t, diskIODescriptor, resourceMetrics)
	internal.AssertDescriptorEqual(t, diskIODescriptor, diskIOMetric)
	if startTime != 0 {
		internal.AssertIntSumMetricStartTimeEquals(t, diskIOMetric, startTime)
	}
	internal.AssertIntSumMetricLabelHasValue(t, diskIOMetric, 0, directionLabelName, readDirectionLabelValue)
	internal.AssertIntSumMetricLabelHasValue(t, diskIOMetric, 1, directionLabelName, writeDirectionLabelValue)
}

func assertSameTimeStampForAllMetricsWithinResource(t *testing.T, resourceMetrics pdata.ResourceMetricsSlice) {
	for i := 0; i < resourceMetrics.Len(); i++ {
		ilms := resourceMetrics.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			internal.AssertSameTimeStampForAllMetrics(t, ilms.At(j).Metrics())
		}
	}
}

func getMetric(t *testing.T, descriptor pdata.Metric, rms pdata.ResourceMetricsSlice) pdata.Metric {
	for i := 0; i < rms.Len(); i++ {
		metrics := getMetricSlice(t, rms.At(i))
		for j := 0; j < metrics.Len(); j++ {
			metric := metrics.At(j)
			if metric.Name() == descriptor.Name() {
				return metric
			}
		}
	}

	require.Fail(t, fmt.Sprintf("no metric with name %s was returned", descriptor.Name()))
	return pdata.NewMetric()
}

func getMetricSlice(t *testing.T, rm pdata.ResourceMetrics) pdata.MetricSlice {
	ilms := rm.InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	return ilms.At(0).Metrics()
}

func TestScrapeMetrics_NewError(t *testing.T) {
	skipTestOnUnsupportedOS(t)

	_, err := newProcessScraper(&Config{Include: MatchConfig{Names: []string{"test"}}})
	require.Error(t, err)
	require.Regexp(t, "^error creating process include filters:", err.Error())

	_, err = newProcessScraper(&Config{Exclude: MatchConfig{Names: []string{"test"}}})
	require.Error(t, err)
	require.Regexp(t, "^error creating process exclude filters:", err.Error())
}

func TestScrapeMetrics_GetProcessesError(t *testing.T) {
	skipTestOnUnsupportedOS(t)

	scraper, err := newProcessScraper(&Config{})
	require.NoError(t, err, "Failed to create process scraper: %v", err)

	scraper.getProcessHandles = func() (processHandles, error) { return nil, errors.New("err1") }

	err = scraper.start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to initialize process scraper: %v", err)

	metrics, err := scraper.scrape(context.Background())
	assert.EqualError(t, err, "err1")
	assert.Equal(t, 0, metrics.Len())
	assert.False(t, consumererror.IsPartialScrapeError(err))
}

type processHandlesMock struct {
	handles []*processHandleMock
}

func (p *processHandlesMock) Pid(index int) int32 {
	return 1
}

func (p *processHandlesMock) At(index int) processHandle {
	return p.handles[index]
}

func (p *processHandlesMock) Len() int {
	return len(p.handles)
}

type processHandleMock struct {
	mock.Mock
}

func (p *processHandleMock) Name() (ret string, err error) {
	args := p.MethodCalled("Name")
	return args.String(0), args.Error(1)
}

func (p *processHandleMock) Exe() (string, error) {
	args := p.MethodCalled("Exe")
	return args.String(0), args.Error(1)
}

func (p *processHandleMock) Username() (string, error) {
	args := p.MethodCalled("Username")
	return args.String(0), args.Error(1)
}

func (p *processHandleMock) Cmdline() (string, error) {
	args := p.MethodCalled("Cmdline")
	return args.String(0), args.Error(1)
}

func (p *processHandleMock) CmdlineSlice() ([]string, error) {
	args := p.MethodCalled("CmdlineSlice")
	return args.Get(0).([]string), args.Error(1)
}

func (p *processHandleMock) Times() (*cpu.TimesStat, error) {
	args := p.MethodCalled("Times")
	return args.Get(0).(*cpu.TimesStat), args.Error(1)
}

func (p *processHandleMock) MemoryInfo() (*process.MemoryInfoStat, error) {
	args := p.MethodCalled("MemoryInfo")
	return args.Get(0).(*process.MemoryInfoStat), args.Error(1)
}

func (p *processHandleMock) IOCounters() (*process.IOCountersStat, error) {
	args := p.MethodCalled("IOCounters")
	return args.Get(0).(*process.IOCountersStat), args.Error(1)
}

func newDefaultHandleMock() *processHandleMock {
	handleMock := &processHandleMock{}
	handleMock.On("Username").Return("username", nil)
	handleMock.On("Cmdline").Return("cmdline", nil)
	handleMock.On("CmdlineSlice").Return([]string{"cmdline"}, nil)
	handleMock.On("Times").Return(&cpu.TimesStat{}, nil)
	handleMock.On("MemoryInfo").Return(&process.MemoryInfoStat{}, nil)
	handleMock.On("IOCounters").Return(&process.IOCountersStat{}, nil)
	return handleMock
}

func TestScrapeMetrics_Filtered(t *testing.T) {
	skipTestOnUnsupportedOS(t)

	type testCase struct {
		name          string
		names         []string
		include       []string
		exclude       []string
		expectedNames []string
	}

	testCases := []testCase{
		{
			name:          "No Filter",
			names:         []string{"test1", "test2"},
			include:       []string{"test*"},
			expectedNames: []string{"test1", "test2"},
		},
		{
			name:          "Include All",
			names:         []string{"test1", "test2"},
			include:       []string{"test*"},
			expectedNames: []string{"test1", "test2"},
		},
		{
			name:          "Include One",
			names:         []string{"test1", "test2"},
			include:       []string{"test1"},
			expectedNames: []string{"test1"},
		},
		{
			name:          "Exclude All",
			names:         []string{"test1", "test2"},
			exclude:       []string{"test*"},
			expectedNames: []string{},
		},
		{
			name:          "Include & Exclude",
			names:         []string{"test1", "test2"},
			include:       []string{"test*"},
			exclude:       []string{"test2"},
			expectedNames: []string{"test1"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config := &Config{}

			if len(test.include) > 0 {
				config.Include = MatchConfig{
					Names:  test.include,
					Config: filterset.Config{MatchType: filterset.Regexp},
				}
			}
			if len(test.exclude) > 0 {
				config.Exclude = MatchConfig{
					Names:  test.exclude,
					Config: filterset.Config{MatchType: filterset.Regexp},
				}
			}

			scraper, err := newProcessScraper(config)
			require.NoError(t, err, "Failed to create process scraper: %v", err)
			err = scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize process scraper: %v", err)

			handles := make([]*processHandleMock, 0, len(test.names))
			for _, name := range test.names {
				handleMock := newDefaultHandleMock()
				handleMock.On("Name").Return(name, nil)
				handleMock.On("Exe").Return(name, nil)
				handles = append(handles, handleMock)
			}

			scraper.getProcessHandles = func() (processHandles, error) {
				return &processHandlesMock{handles: handles}, nil
			}

			resourceMetrics, err := scraper.scrape(context.Background())
			require.NoError(t, err)

			assert.Equal(t, len(test.expectedNames), resourceMetrics.Len())
			for i, expectedName := range test.expectedNames {
				rm := resourceMetrics.At(i)
				name, _ := rm.Resource().Attributes().Get(conventions.AttributeProcessExecutableName)
				assert.Equal(t, expectedName, name.StringVal())
			}
		})
	}
}

func TestScrapeMetrics_ProcessErrors(t *testing.T) {
	skipTestOnUnsupportedOS(t)

	type testCase struct {
		name            string
		osFilter        string
		nameError       error
		exeError        error
		usernameError   error
		cmdlineError    error
		timesError      error
		memoryInfoError error
		ioCountersError error
		expectedError   string
	}

	testCases := []testCase{
		{
			name:          "Name Error",
			osFilter:      "windows",
			nameError:     errors.New("err1"),
			expectedError: `error reading process name for pid 1: err1`,
		},
		{
			name:          "Exe Error",
			exeError:      errors.New("err1"),
			expectedError: `error reading process name for pid 1: err1`,
		},
		{
			name:          "Cmdline Error",
			cmdlineError:  errors.New("err2"),
			expectedError: `error reading command for process "test" (pid 1): err2`,
		},
		{
			name:          "Username Error",
			usernameError: errors.New("err3"),
			expectedError: `error reading username for process "test" (pid 1): err3`,
		},
		{
			name:          "Times Error",
			timesError:    errors.New("err4"),
			expectedError: `error reading cpu times for process "test" (pid 1): err4`,
		},
		{
			name:            "Memory Info Error",
			memoryInfoError: errors.New("err5"),
			expectedError:   `error reading memory info for process "test" (pid 1): err5`,
		},
		{
			name:            "IO Counters Error",
			ioCountersError: errors.New("err6"),
			expectedError:   `error reading disk usage for process "test" (pid 1): err6`,
		},
		{
			name:            "Multiple Errors",
			cmdlineError:    errors.New("err2"),
			usernameError:   errors.New("err3"),
			timesError:      errors.New("err4"),
			memoryInfoError: errors.New("err5"),
			ioCountersError: errors.New("err6"),
			expectedError: `[[error reading command for process "test" (pid 1): err2; ` +
				`error reading username for process "test" (pid 1): err3]; ` +
				`error reading cpu times for process "test" (pid 1): err4; ` +
				`error reading memory info for process "test" (pid 1): err5; ` +
				`error reading disk usage for process "test" (pid 1): err6]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.osFilter == runtime.GOOS {
				t.Skipf("skipping test %v on %v", test.name, runtime.GOOS)
			}

			scraper, err := newProcessScraper(&Config{})
			require.NoError(t, err, "Failed to create process scraper: %v", err)
			err = scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize process scraper: %v", err)

			username := "username"
			if test.usernameError != nil {
				username = ""
			}

			handleMock := &processHandleMock{}
			handleMock.On("Name").Return("test", test.nameError)
			handleMock.On("Exe").Return("test", test.exeError)
			handleMock.On("Username").Return(username, test.usernameError)
			handleMock.On("Cmdline").Return("cmdline", test.cmdlineError)
			handleMock.On("CmdlineSlice").Return([]string{"cmdline"}, test.cmdlineError)
			handleMock.On("Times").Return(&cpu.TimesStat{}, test.timesError)
			handleMock.On("MemoryInfo").Return(&process.MemoryInfoStat{}, test.memoryInfoError)
			handleMock.On("IOCounters").Return(&process.IOCountersStat{}, test.ioCountersError)

			scraper.getProcessHandles = func() (processHandles, error) {
				return &processHandlesMock{handles: []*processHandleMock{handleMock}}, nil
			}

			resourceMetrics, err := scraper.scrape(context.Background())

			md := pdata.NewMetrics()
			resourceMetrics.MoveAndAppendTo(md.ResourceMetrics())
			expectedResourceMetricsLen, expectedMetricsLen := getExpectedLengthOfReturnedMetrics(test.nameError, test.exeError, test.timesError, test.memoryInfoError, test.ioCountersError)
			assert.Equal(t, expectedResourceMetricsLen, md.ResourceMetrics().Len())
			assert.Equal(t, expectedMetricsLen, md.MetricCount())

			assert.EqualError(t, err, test.expectedError)
			isPartial := consumererror.IsPartialScrapeError(err)
			assert.True(t, isPartial)
			if isPartial {
				expectedFailures := getExpectedScrapeFailures(test.nameError, test.exeError, test.timesError, test.memoryInfoError, test.ioCountersError)
				assert.Equal(t, expectedFailures, err.(consumererror.PartialScrapeError).Failed)
			}
		})
	}
}

func getExpectedLengthOfReturnedMetrics(nameError, exeError, timeError, memError, diskError error) (int, int) {
	if nameError != nil || exeError != nil {
		return 0, 0
	}

	expectedLen := 0
	if timeError == nil {
		expectedLen += cpuMetricsLen
	}
	if memError == nil {
		expectedLen += memoryMetricsLen
	}
	if diskError == nil {
		expectedLen += diskMetricsLen
	}
	return 1, expectedLen
}

func getExpectedScrapeFailures(nameError, exeError, timeError, memError, diskError error) int {
	expectedResourceMetricsLen, expectedMetricsLen := getExpectedLengthOfReturnedMetrics(nameError, exeError, timeError, memError, diskError)
	if expectedResourceMetricsLen == 0 {
		return 1
	}

	return metricsLen - expectedMetricsLen
}
