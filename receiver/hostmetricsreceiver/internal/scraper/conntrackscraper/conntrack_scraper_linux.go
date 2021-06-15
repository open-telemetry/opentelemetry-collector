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

// +build linux

package conntrackscraper

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const metricsLen = 2

func scrapeAndAppendConntrackMetrics(s *scraper, metrics pdata.MetricSlice) error {
	now := pdata.TimestampFromTime(time.Now())

	conntrack, err := s.conntrack()
	if err != nil {
		return fmt.Errorf("failed to scrape conntrack metrics: %w", err)
	}

	if len(conntrack) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + metricsLen)
		initializeNetworkConntrackMetric(metrics.At(startIdx+0), metadata.Metrics.SystemConntrackCount, now, conntrack[0].ConnTrackCount)
		initializeNetworkConntrackMetric(metrics.At(startIdx+1), metadata.Metrics.SystemConntrackMax, now, conntrack[0].ConnTrackMax)
	}
	return nil
}

func initializeNetworkConntrackMetric(metric pdata.Metric, metricIntf metadata.MetricIntf, now pdata.Timestamp, value int64) {
	metricIntf.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(1)
	initializeNetworkConntrackDataPoint(idps.At(0), now, value)
}

func initializeNetworkConntrackDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, value int64) {
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
