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
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for Network Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano

	// for mocking
	bootTime    func() (uint64, error)
	ioCounters  func(bool) ([]net.IOCountersStat, error)
	connections func(string) ([]net.ConnectionStat, error)
}

// newNetworkScraper creates a set of Network related metrics
func newNetworkScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, bootTime: host.BootTime, ioCounters: net.IOCounters, connections: net.Connections}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime * 1e9)
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	var errors []error

	err := s.scrapeAndAppendNetworkCounterMetrics(metrics, s.startTime)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendNetworkTCPConnectionsMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

func (s *scraper) scrapeAndAppendNetworkCounterMetrics(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano) error {
	// get total stats only
	networkStatsSlice, err := s.ioCounters( /*perNetworkInterfaceController=*/ false)
	if err != nil {
		return err
	}

	networkStats := networkStatsSlice[0]

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 4)
	initializeNetworkMetric(metrics.At(startIdx+0), networkPacketsDescriptor, startTime, networkStats.PacketsSent, networkStats.PacketsRecv)
	initializeNetworkMetric(metrics.At(startIdx+1), networkDroppedPacketsDescriptor, startTime, networkStats.Dropout, networkStats.Dropin)
	initializeNetworkMetric(metrics.At(startIdx+2), networkErrorsDescriptor, startTime, networkStats.Errout, networkStats.Errin)
	initializeNetworkMetric(metrics.At(startIdx+3), networkIODescriptor, startTime, networkStats.BytesSent, networkStats.BytesRecv)
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

func (s *scraper) scrapeAndAppendNetworkTCPConnectionsMetric(metrics pdata.MetricSlice) error {
	connections, err := s.connections("tcp")
	if err != nil {
		return err
	}

	connectionStatusCounts := getTCPConnectionStatusCounts(connections)

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeNetworkTCPConnectionsMetric(metrics.At(startIdx), connectionStatusCounts)
	return nil
}

func getTCPConnectionStatusCounts(connections []net.ConnectionStat) map[string]int64 {
	var tcpStatuses = map[string]int64{
		"CLOSE_WAIT":   0,
		"CLOSED":       0,
		"CLOSING":      0,
		"DELETE":       0,
		"ESTABLISHED":  0,
		"FIN_WAIT_1":   0,
		"FIN_WAIT_2":   0,
		"LAST_ACK":     0,
		"LISTEN":       0,
		"SYN_SENT":     0,
		"SYN_RECEIVED": 0,
		"TIME_WAIT":    0,
	}

	for _, connection := range connections {
		tcpStatuses[connection.Status]++
	}
	return tcpStatuses
}

func initializeNetworkTCPConnectionsMetric(metric pdata.Metric, connectionStateCounts map[string]int64) {
	networkTCPConnectionsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(connectionStateCounts))

	i := 0
	for connectionState, count := range connectionStateCounts {
		initializeNetworkTCPConnectionsDataPoint(idps.At(i), connectionState, count)
		i++
	}
}

func initializeNetworkTCPConnectionsDataPoint(dataPoint pdata.Int64DataPoint, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
