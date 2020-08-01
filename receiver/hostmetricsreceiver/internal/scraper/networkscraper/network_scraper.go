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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// scraper for Network Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	// for mocking
	bootTime    func() (uint64, error)
	ioCounters  func(bool) ([]net.IOCountersStat, error)
	connections func(string) ([]net.ConnectionStat, error)
}

// newNetworkScraper creates a set of Network related metrics
func newNetworkScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, bootTime: host.BootTime, ioCounters: net.IOCounters, connections: net.Connections}

	var err error

	if len(cfg.Include.Interfaces) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Interfaces, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating network interface include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Interfaces) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Interfaces, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating network interface exclude filters: %w", err)
		}
	}

	return scraper, nil
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

	return metrics, componenterror.CombineErrors(errors)
}

func (s *scraper) scrapeAndAppendNetworkCounterMetrics(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano) error {
	// get total stats only
	ioCounters, err := s.ioCounters( /*perNetworkInterfaceController=*/ true)
	if err != nil {
		return err
	}

	// filter network interfaces by name
	ioCounters = s.filterByInterface(ioCounters)

	if len(ioCounters) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + 4)
		initializeNetworkPacketsMetric(metrics.At(startIdx+0), networkPacketsDescriptor, startTime, ioCounters)
		initializeNetworkDroppedPacketsMetric(metrics.At(startIdx+1), networkDroppedPacketsDescriptor, startTime, ioCounters)
		initializeNetworkErrorsMetric(metrics.At(startIdx+2), networkErrorsDescriptor, startTime, ioCounters)
		initializeNetworkIOMetric(metrics.At(startIdx+3), networkIODescriptor, startTime, ioCounters)
	}

	return nil
}

func initializeNetworkPacketsMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, ioCountersSlice []net.IOCountersStat) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCountersSlice))
	for idx, ioCounters := range ioCountersSlice {
		initializeNetworkDataPoint(idps.At(2*idx+0), startTime, ioCounters.Name, transmitDirectionLabelValue, int64(ioCounters.PacketsSent))
		initializeNetworkDataPoint(idps.At(2*idx+1), startTime, ioCounters.Name, receiveDirectionLabelValue, int64(ioCounters.PacketsRecv))
	}
}

func initializeNetworkDroppedPacketsMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, ioCountersSlice []net.IOCountersStat) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCountersSlice))
	for idx, ioCounters := range ioCountersSlice {
		initializeNetworkDataPoint(idps.At(2*idx+0), startTime, ioCounters.Name, transmitDirectionLabelValue, int64(ioCounters.Dropout))
		initializeNetworkDataPoint(idps.At(2*idx+1), startTime, ioCounters.Name, receiveDirectionLabelValue, int64(ioCounters.Dropin))
	}
}

func initializeNetworkErrorsMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, ioCountersSlice []net.IOCountersStat) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCountersSlice))
	for idx, ioCounters := range ioCountersSlice {
		initializeNetworkDataPoint(idps.At(2*idx+0), startTime, ioCounters.Name, transmitDirectionLabelValue, int64(ioCounters.Errout))
		initializeNetworkDataPoint(idps.At(2*idx+1), startTime, ioCounters.Name, receiveDirectionLabelValue, int64(ioCounters.Errin))
	}
}

func initializeNetworkIOMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, ioCountersSlice []net.IOCountersStat) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCountersSlice))
	for idx, ioCounters := range ioCountersSlice {
		initializeNetworkDataPoint(idps.At(2*idx+0), startTime, ioCounters.Name, transmitDirectionLabelValue, int64(ioCounters.BytesSent))
		initializeNetworkDataPoint(idps.At(2*idx+1), startTime, ioCounters.Name, receiveDirectionLabelValue, int64(ioCounters.BytesRecv))
	}
}

func initializeNetworkDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, interfaceLabel, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(interfaceLabelName, interfaceLabel)
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

func (s *scraper) filterByInterface(ioCounters []net.IOCountersStat) []net.IOCountersStat {
	if s.includeFS == nil && s.excludeFS == nil {
		return ioCounters
	}

	filteredIOCounters := make([]net.IOCountersStat, 0, len(ioCounters))
	for _, io := range ioCounters {
		if s.includeInterface(io.Name) {
			filteredIOCounters = append(filteredIOCounters, io)
		}
	}
	return filteredIOCounters
}

func (s *scraper) includeInterface(interfaceName string) bool {
	return (s.includeFS == nil || s.includeFS.Matches(interfaceName)) &&
		(s.excludeFS == nil || !s.excludeFS.Matches(interfaceName))
}
