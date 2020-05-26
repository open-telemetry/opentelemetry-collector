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

package networkscraper

import (
	"context"
	"strings"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for Network Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
}

// newNetworkScraper creates a set of Network related metrics
func newNetworkScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime)
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(ctx context.Context) (pdata.MetricSlice, error) {
	_, span := trace.StartSpan(ctx, "networkscraper.ScrapeMetrics")
	defer span.End()

	metrics := pdata.NewMetricSlice()

	var errors []error

	err := scrapeAndAppendNetworkCounterMetrics(metrics, s.startTime)
	if err != nil {
		errors = append(errors, err)
	}

	err = scrapeAndAppendNetworkTCPConnectionsMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

func scrapeAndAppendNetworkCounterMetrics(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano) error {
	// get total stats only
	perNetworkInterfaceController := false

	networkStatsSlice, err := net.IOCounters(perNetworkInterfaceController)
	if err != nil {
		return err
	}

	networkStats := networkStatsSlice[0]

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 4)
	initializeNetworkMetric(metrics.At(startIdx+0), metricNetworkPacketsDescriptor, startTime, networkStats.PacketsSent, networkStats.PacketsRecv)
	initializeNetworkMetric(metrics.At(startIdx+1), metricNetworkDroppedPacketsDescriptor, startTime, networkStats.Dropout, networkStats.Dropin)
	initializeNetworkMetric(metrics.At(startIdx+2), metricNetworkErrorsDescriptor, startTime, networkStats.Errout, networkStats.Errin)
	initializeNetworkMetric(metrics.At(startIdx+3), metricNetworkBytesDescriptor, startTime, networkStats.BytesSent, networkStats.BytesRecv)
	return nil
}

func initializeNetworkMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, transmitValue, receiveValue uint64) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2)
	initializeNetworkDataPoint(idps.At(0), startTime, transmitDirectionLabelValue, int64(transmitValue))
	initializeNetworkDataPoint(idps.At(1), startTime, receiveDirectionLabelValue, int64(receiveValue))
}

func initializeNetworkDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}

func scrapeAndAppendNetworkTCPConnectionsMetric(metrics pdata.MetricSlice) error {
	connections, err := net.Connections("tcp")
	if err != nil {
		return err
	}

	connectionStatusCounts := getConnectionStatusCounts(connections)

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeNetworkTCPConnectionsMetric(metrics.At(startIdx), connectionStatusCounts)
	return nil
}

func getConnectionStatusCounts(connections []net.ConnectionStat) map[string]int64 {
	connectionStatuses := make(map[string]int64, len(connections))
	for _, connection := range connections {
		connectionStatuses[strings.ToLower(connection.Status)]++
	}
	return connectionStatuses
}

func initializeNetworkTCPConnectionsMetric(metric pdata.Metric, connectionStateCounts map[string]int64) {
	metricNetworkTCPConnectionDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(connectionStateCounts))

	i := 0
	for connectionState, count := range connectionStateCounts {
		initializeTCPConnectionsDataPoint(idps.At(i), connectionState, count)
		i++
	}
}

func initializeTCPConnectionsDataPoint(dataPoint pdata.Int64DataPoint, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
