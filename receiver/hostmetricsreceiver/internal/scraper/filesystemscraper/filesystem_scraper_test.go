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

package filesystemscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name                      string
		config                    Config
		partitionsFunc            func(bool) ([]disk.PartitionStat, error)
		usageFunc                 func(string) (*disk.UsageStat, error)
		expectMetrics             bool
		expectedDeviceDataPoints  int
		expectedDeviceLabelValues []map[string]string
		newErrRegex               string
		expectedErr               string
	}

	testCases := []testCase{
		{
			name:          "Standard",
			expectMetrics: true,
		},
		{
			name:   "Include single device filter",
			config: Config{IncludeDevices: DeviceMatchConfig{filterset.Config{MatchType: "strict"}, []string{"a"}}},
			partitionsFunc: func(bool) ([]disk.PartitionStat, error) {
				return []disk.PartitionStat{{Device: "a"}, {Device: "b"}}, nil
			},
			usageFunc: func(string) (*disk.UsageStat, error) {
				return &disk.UsageStat{}, nil
			},
			expectMetrics:            true,
			expectedDeviceDataPoints: 1,
		},
		{
			name:          "Include Device Filter that matches nothing",
			config:        Config{IncludeDevices: DeviceMatchConfig{filterset.Config{MatchType: "strict"}, []string{"@*^#&*$^#)"}}},
			expectMetrics: false,
		},
		{
			name: "Include filter with devices, filesystem type and mount points",
			config: Config{
				IncludeDevices: DeviceMatchConfig{
					Config: filterset.Config{
						MatchType: filterset.Strict,
					},
					Devices: []string{"device_a", "device_b"},
				},
				ExcludeFSTypes: FSTypeMatchConfig{
					Config: filterset.Config{
						MatchType: filterset.Strict,
					},
					FSTypes: []string{"fs_type_b"},
				},
				ExcludeMountPoints: MountPointMatchConfig{
					Config: filterset.Config{
						MatchType: filterset.Strict,
					},
					MountPoints: []string{"mount_point_b", "mount_point_c"},
				},
			},
			usageFunc: func(s string) (*disk.UsageStat, error) {
				return &disk.UsageStat{
					Fstype: "fs_type_a",
				}, nil
			},
			partitionsFunc: func(b bool) ([]disk.PartitionStat, error) {
				return []disk.PartitionStat{
					{
						Device:     "device_a",
						Mountpoint: "mount_point_a",
						Fstype:     "fs_type_a",
					},
					{
						Device:     "device_a",
						Mountpoint: "mount_point_b",
						Fstype:     "fs_type_b",
					},
					{
						Device:     "device_b",
						Mountpoint: "mount_point_c",
						Fstype:     "fs_type_b",
					},
					{
						Device:     "device_b",
						Mountpoint: "mount_point_d",
						Fstype:     "fs_type_c",
					},
				}, nil
			},
			expectMetrics:            true,
			expectedDeviceDataPoints: 2,
			expectedDeviceLabelValues: []map[string]string{
				{
					"device":     "device_a",
					"mountpoint": "mount_point_a",
					"type":       "fs_type_a",
					"mode":       "unknown",
				},
				{
					"device":     "device_b",
					"mountpoint": "mount_point_d",
					"type":       "fs_type_c",
					"mode":       "unknown",
				},
			},
		},
		{
			name:        "Invalid Include Device Filter",
			config:      Config{IncludeDevices: DeviceMatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device include filters:",
		},
		{
			name:        "Invalid Exclude Device Filter",
			config:      Config{ExcludeDevices: DeviceMatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device exclude filters:",
		},
		{
			name:        "Invalid Include Filesystems Filter",
			config:      Config{IncludeFSTypes: FSTypeMatchConfig{FSTypes: []string{"test"}}},
			newErrRegex: "^error creating type include filters:",
		},
		{
			name:        "Invalid Exclude Filesystems Filter",
			config:      Config{ExcludeFSTypes: FSTypeMatchConfig{FSTypes: []string{"test"}}},
			newErrRegex: "^error creating type exclude filters:",
		},
		{
			name:        "Invalid Include Moountpoints Filter",
			config:      Config{IncludeMountPoints: MountPointMatchConfig{MountPoints: []string{"test"}}},
			newErrRegex: "^error creating mountpoint include filters:",
		},
		{
			name:        "Invalid Exclude Moountpoints Filter",
			config:      Config{ExcludeMountPoints: MountPointMatchConfig{MountPoints: []string{"test"}}},
			newErrRegex: "^error creating mountpoint exclude filters:",
		},
		{
			name:           "Partitions Error",
			partitionsFunc: func(bool) ([]disk.PartitionStat, error) { return nil, errors.New("err1") },
			expectedErr:    "err1",
		},
		{
			name:        "Usage Error",
			usageFunc:   func(string) (*disk.UsageStat, error) { return nil, errors.New("err2") },
			expectedErr: "err2",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newFileSystemScraper(context.Background(), &test.config)
			if test.newErrRegex != "" {
				require.Error(t, err)
				require.Regexp(t, test.newErrRegex, err)
				return
			}
			require.NoError(t, err, "Failed to create file system scraper: %v", err)

			if test.partitionsFunc != nil {
				scraper.partitions = test.partitionsFunc
			}
			if test.usageFunc != nil {
				scraper.usage = test.usageFunc
			}

			metrics, err := scraper.Scrape(context.Background())
			if test.expectedErr != "" {
				assert.Contains(t, err.Error(), test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, metricsLen, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			if !test.expectMetrics {
				assert.Equal(t, 0, metrics.Len())
				return
			}

			assert.GreaterOrEqual(t, metrics.Len(), 1)

			assertFileSystemUsageMetricValid(
				t,
				metrics.At(0),
				metadata.Metrics.SystemFilesystemUsage.New(),
				test.expectedDeviceDataPoints*fileSystemStatesLen,
				test.expectedDeviceLabelValues,
			)

			if isUnix() {
				assertFileSystemUsageMetricHasUnixSpecificStateLabels(t, metrics.At(0))
				assertFileSystemUsageMetricValid(
					t,
					metrics.At(1),
					metadata.Metrics.SystemFilesystemInodesUsage.New(),
					test.expectedDeviceDataPoints*2,
					test.expectedDeviceLabelValues,
				)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertFileSystemUsageMetricValid(
	t *testing.T,
	metric pdata.Metric,
	descriptor pdata.Metric,
	expectedDeviceDataPoints int,
	expectedDeviceLabelValues []map[string]string) {
	internal.AssertDescriptorEqual(t, descriptor, metric)
	for i := 0; i < metric.IntSum().DataPoints().Len(); i++ {
		for _, label := range []string{"device", "type", "mode", "mountpoint"} {
			internal.AssertIntSumMetricLabelExists(t, metric, i, label)
		}
	}

	if expectedDeviceDataPoints > 0 {
		assert.Equal(t, expectedDeviceDataPoints, metric.IntSum().DataPoints().Len())

		// Assert label values if specified.
		if expectedDeviceLabelValues != nil {
			dpsPerDevice := expectedDeviceDataPoints / len(expectedDeviceLabelValues)
			deviceIdx := 0
			for i := 0; i < metric.IntSum().DataPoints().Len(); i += dpsPerDevice {
				for j := i; j < i+dpsPerDevice; j++ {
					for labelKey, labelValue := range expectedDeviceLabelValues[deviceIdx] {
						internal.AssertIntSumMetricLabelHasValue(t, metric, j, labelKey, labelValue)
					}
				}
				deviceIdx++
			}
		}
	} else {
		assert.GreaterOrEqual(t, metric.IntSum().DataPoints().Len(), fileSystemStatesLen)
	}
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, "state", "used")
	internal.AssertIntSumMetricLabelHasValue(t, metric, 1, "state", "free")
}

func assertFileSystemUsageMetricHasUnixSpecificStateLabels(t *testing.T, metric pdata.Metric) {
	internal.AssertIntSumMetricLabelHasValue(t, metric, 2, "state", "reserved")
}

func isUnix() bool {
	for _, unixOS := range []string{"linux", "darwin", "freebsd", "openbsd", "solaris"} {
		if runtime.GOOS == unixOS {
			return true
		}
	}

	return false
}
