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

package diskscraper

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/third_party/telegraf/win_perf_counters"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/windows/pdh"
)

const (
	logicalDiskReadsPerSecPath  = `\LogicalDisk(*)\Disk Reads/sec`
	logicalDiskWritesPerSecPath = `\LogicalDisk(*)\Disk Writes/sec`

	logicalDiskReadBytesPerSecPath  = `\LogicalDisk(*)\Disk Read Bytes/sec`
	logicalDiskWriteBytesPerSecPath = `\LogicalDisk(*)\Disk Write Bytes/sec`

	logicalAvgDiskSecsPerReadPath  = `\LogicalDisk(*)\Avg. Disk sec/Read`
	logicalAvgDiskSecsPerWritePath = `\LogicalDisk(*)\Avg. Disk sec/Write`

	logicalDiskQueueLengthPath = `\LogicalDisk(*)\Current Disk Queue Length`
)

// scraper for Disk Metrics
type scraper struct {
	config         *Config
	startTime      pdata.TimestampUnixNano
	prevScrapeTime time.Time
	includeFS      filterset.FilterSet
	excludeFS      filterset.FilterSet

	diskReadBytesPerSecCounter  pdh.PerfCounterScraper
	diskWriteBytesPerSecCounter pdh.PerfCounterScraper
	diskReadsPerSecCounter      pdh.PerfCounterScraper
	diskWritesPerSecCounter     pdh.PerfCounterScraper
	avgDiskSecsPerReadCounter   pdh.PerfCounterScraper
	avgDiskSecsPerWriteCounter  pdh.PerfCounterScraper
	diskQueueLengthCounter      pdh.PerfCounterScraper

	cumulativeDiskIO   cumulativeDiskValues
	cumulativeDiskOps  cumulativeDiskValues
	cumulativeDiskTime cumulativeDiskValues
}

type cumulativeDiskValues map[string]*value

func (cv cumulativeDiskValues) getOrAdd(k string) *value {
	if v, ok := cv[k]; ok {
		return v
	}

	v := &value{}
	cv[k] = v
	return v
}

type value struct {
	read  float64
	write float64
}

// newDiskScraper creates a Disk Scraper
func newDiskScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{
		config:             cfg,
		cumulativeDiskIO:   map[string]*value{},
		cumulativeDiskOps:  map[string]*value{},
		cumulativeDiskTime: map[string]*value{},
	}

	var err error

	if len(cfg.Include.Devices) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Devices, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Devices) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Devices, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device exclude filters: %w", err)
		}
	}

	return scraper, nil
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	s.startTime = internal.TimeToUnixNano(time.Now())
	s.prevScrapeTime = time.Now()

	var err error

	s.diskReadBytesPerSecCounter, err = pdh.NewPerfCounter(logicalDiskReadBytesPerSecPath, true)
	if err != nil {
		return err
	}

	s.diskWriteBytesPerSecCounter, err = pdh.NewPerfCounter(logicalDiskWriteBytesPerSecPath, true)
	if err != nil {
		return err
	}

	s.diskReadsPerSecCounter, err = pdh.NewPerfCounter(logicalDiskReadsPerSecPath, true)
	if err != nil {
		return err
	}

	s.diskWritesPerSecCounter, err = pdh.NewPerfCounter(logicalDiskWritesPerSecPath, true)
	if err != nil {
		return err
	}

	s.avgDiskSecsPerReadCounter, err = pdh.NewPerfCounter(logicalAvgDiskSecsPerReadPath, true)
	if err != nil {
		return err
	}

	s.avgDiskSecsPerWriteCounter, err = pdh.NewPerfCounter(logicalAvgDiskSecsPerWritePath, true)
	if err != nil {
		return err
	}

	s.diskQueueLengthCounter, err = pdh.NewPerfCounter(logicalDiskQueueLengthPath, true)
	if err != nil {
		return err
	}

	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	var errors []error

	if s.diskReadBytesPerSecCounter != nil && !reflect.ValueOf(s.diskReadBytesPerSecCounter).IsNil() {
		if err := s.diskReadBytesPerSecCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.diskWriteBytesPerSecCounter != nil && !reflect.ValueOf(s.diskWriteBytesPerSecCounter).IsNil() {
		if err := s.diskWriteBytesPerSecCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.diskReadsPerSecCounter != nil && !reflect.ValueOf(s.diskReadsPerSecCounter).IsNil() {
		if err := s.diskReadsPerSecCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.diskWritesPerSecCounter != nil && !reflect.ValueOf(s.diskWritesPerSecCounter).IsNil() {
		if err := s.diskWritesPerSecCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.avgDiskSecsPerReadCounter != nil && !reflect.ValueOf(s.avgDiskSecsPerReadCounter).IsNil() {
		if err := s.avgDiskSecsPerReadCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.avgDiskSecsPerWriteCounter != nil && !reflect.ValueOf(s.avgDiskSecsPerWriteCounter).IsNil() {
		if err := s.avgDiskSecsPerWriteCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if s.diskQueueLengthCounter != nil && !reflect.ValueOf(s.diskQueueLengthCounter).IsNil() {
		if err := s.diskQueueLengthCounter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	return componenterror.CombineErrors(errors)
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (dataold.MetricSlice, error) {
	now := time.Now()
	durationSinceLastScraped := now.Sub(s.prevScrapeTime).Seconds()
	s.prevScrapeTime = now
	nowUnixTime := pdata.TimestampUnixNano(uint64(now.UnixNano()))

	metrics := dataold.NewMetricSlice()

	var errors []error

	err := s.scrapeAndAppendDiskIOMetric(metrics, nowUnixTime, durationSinceLastScraped)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendDiskOpsMetric(metrics, nowUnixTime, durationSinceLastScraped)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendDiskPendingOperationsMetric(metrics, nowUnixTime)
	if err != nil {
		errors = append(errors, err)
	}

	return metrics, componenterror.CombineErrors(errors)
}

func (s *scraper) scrapeAndAppendDiskIOMetric(metrics dataold.MetricSlice, now pdata.TimestampUnixNano, durationSinceLastScraped float64) error {
	diskReadBytesPerSecValues, err := s.diskReadBytesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	diskWriteBytesPerSecValues, err := s.diskWriteBytesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	for _, diskReadBytesPerSec := range diskReadBytesPerSecValues {
		if s.includeDevice(diskReadBytesPerSec.InstanceName) {
			s.cumulativeDiskIO.getOrAdd(diskReadBytesPerSec.InstanceName).read += diskReadBytesPerSec.Value * durationSinceLastScraped
		}
	}

	for _, diskWriteBytesPerSec := range diskWriteBytesPerSecValues {
		if s.includeDevice(diskWriteBytesPerSec.InstanceName) {
			s.cumulativeDiskIO.getOrAdd(diskWriteBytesPerSec.InstanceName).write += diskWriteBytesPerSec.Value * durationSinceLastScraped
		}
	}

	if len(s.cumulativeDiskIO) == 0 {
		return nil
	}

	idx := metrics.Len()
	metrics.Resize(idx + 1)
	initializeDiskInt64Metric(metrics.At(idx), diskIODescriptor, s.startTime, now, s.cumulativeDiskIO)
	return nil
}

func (s *scraper) scrapeAndAppendDiskOpsMetric(metrics dataold.MetricSlice, now pdata.TimestampUnixNano, durationSinceLastScraped float64) error {
	diskReadsPerSecValues, err := s.diskReadsPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	diskWritesPerSecValues, err := s.diskWritesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	avgDiskSecsPerReadValues, err := s.avgDiskSecsPerReadCounter.ScrapeData()
	if err != nil {
		return err
	}
	avgDiskSecsPerReadMap := toMap(avgDiskSecsPerReadValues)

	avgDiskSecsPerWriteValues, err := s.avgDiskSecsPerWriteCounter.ScrapeData()
	if err != nil {
		return err
	}
	avgDiskSecsPerWriteMap := toMap(avgDiskSecsPerWriteValues)

	for _, diskReadsPerSec := range diskReadsPerSecValues {
		device := diskReadsPerSec.InstanceName
		if !s.includeDevice(device) {
			continue
		}

		deltaReadOperations := diskReadsPerSec.Value * durationSinceLastScraped
		s.cumulativeDiskOps.getOrAdd(device).read += deltaReadOperations
		if avgDiskSecsPerRead, ok := avgDiskSecsPerReadMap[device]; ok {
			s.cumulativeDiskTime.getOrAdd(device).read += deltaReadOperations * avgDiskSecsPerRead
		}
	}

	for _, diskWritesPerSec := range diskWritesPerSecValues {
		device := diskWritesPerSec.InstanceName
		if !s.includeDevice(device) {
			continue
		}

		deltaWriteOperations := diskWritesPerSec.Value * durationSinceLastScraped
		s.cumulativeDiskOps.getOrAdd(device).write += deltaWriteOperations
		if avgDiskSecsPerWrite, ok := avgDiskSecsPerWriteMap[device]; ok {
			s.cumulativeDiskTime.getOrAdd(device).write += deltaWriteOperations * avgDiskSecsPerWrite
		}
	}

	idx := metrics.Len()

	if len(s.cumulativeDiskIO) > 0 {
		metrics.Resize(idx + 1)
		initializeDiskInt64Metric(metrics.At(idx), diskOpsDescriptor, s.startTime, now, s.cumulativeDiskOps)
		idx++
	}

	if len(s.cumulativeDiskTime) > 0 {
		metrics.Resize(idx + 1)
		initializeDiskDoubleMetric(metrics.At(idx), diskTimeDescriptor, s.startTime, now, s.cumulativeDiskTime)
		idx++
	}

	return nil
}

func (s *scraper) scrapeAndAppendDiskPendingOperationsMetric(metrics dataold.MetricSlice, now pdata.TimestampUnixNano) error {
	diskQueueLengthValues, err := s.diskQueueLengthCounter.ScrapeData()
	if err != nil {
		return err
	}

	filteredDiskQueueLengthValues := s.filterByDevice(diskQueueLengthValues)
	if len(filteredDiskQueueLengthValues) == 0 {
		return nil
	}

	idx := metrics.Len()
	metrics.Resize(idx + 1)
	initializeDiskPendingOperationsMetric(metrics.At(idx), now, filteredDiskQueueLengthValues)
	return nil
}

func initializeDiskInt64Metric(metric dataold.Metric, descriptor dataold.MetricDescriptor, startTime, now pdata.TimestampUnixNano, ops cumulativeDiskValues) {
	descriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ops))

	idx := 0
	for device, value := range ops {
		initializeInt64DataPoint(idps.At(idx+0), startTime, now, device, readDirectionLabelValue, int64(value.read))
		initializeInt64DataPoint(idps.At(idx+1), startTime, now, device, writeDirectionLabelValue, int64(value.write))
		idx += 2
	}
}

func initializeDiskDoubleMetric(metric dataold.Metric, descriptor dataold.MetricDescriptor, startTime, now pdata.TimestampUnixNano, ops cumulativeDiskValues) {
	descriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(2 * len(ops))

	idx := 0
	for device, value := range ops {
		initializeDoubleDataPoint(ddps.At(idx+0), startTime, now, device, readDirectionLabelValue, value.read)
		initializeDoubleDataPoint(ddps.At(idx+1), startTime, now, device, writeDirectionLabelValue, value.write)
		idx += 2
	}
}

func initializeDiskPendingOperationsMetric(metric dataold.Metric, now pdata.TimestampUnixNano, avgDiskQueueLengthValues []win_perf_counters.CounterValue) {
	diskPendingOperationsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(avgDiskQueueLengthValues))

	for idx, avgDiskQueueLengthValue := range avgDiskQueueLengthValues {
		initializeDiskPendingDataPoint(idps.At(idx), now, avgDiskQueueLengthValue.InstanceName, int64(avgDiskQueueLengthValue.Value))
	}
}

func initializeInt64DataPoint(dataPoint dataold.Int64DataPoint, startTime, now pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDoubleDataPoint(dataPoint dataold.DoubleDataPoint, startTime, now pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDiskPendingDataPoint(dataPoint dataold.Int64DataPoint, now pdata.TimestampUnixNano, deviceLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) filterByDevice(values []win_perf_counters.CounterValue) []win_perf_counters.CounterValue {
	if s.includeFS == nil && s.excludeFS == nil {
		return values
	}

	filteredValues := make([]win_perf_counters.CounterValue, 0, len(values))
	for _, value := range values {
		if s.includeDevice(value.InstanceName) {
			filteredValues = append(filteredValues, value)
		}
	}
	return filteredValues
}

func (s *scraper) includeDevice(deviceName string) bool {
	return (s.includeFS == nil || s.includeFS.Matches(deviceName)) &&
		(s.excludeFS == nil || !s.excludeFS.Matches(deviceName))
}

func toMap(values []win_perf_counters.CounterValue) map[string]float64 {
	mp := make(map[string]float64, len(values))
	for _, value := range values {
		mp[value.InstanceName] = value.Value
	}
	return mp
}
