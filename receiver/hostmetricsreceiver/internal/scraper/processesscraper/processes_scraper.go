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

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

const metricsLen = systemSpecificMetricsLen

// scraper for Processes Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano

	// for mocking gopsutil load.Misc
	misc getMiscStats
}

type getMiscStats func() (*load.MiscStat, error)

// newProcessesScraper creates a set of Processes related metrics
func newProcessesScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, misc: load.Misc}
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime)
	return nil
}

func (s *scraper) scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()
	err := appendSystemSpecificProcessesMetrics(metrics, 0, s.misc)
	return metrics, err
}
