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

package diskscraper

import (
	"context"
	"reflect"
	"time"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/windows/pdh"
)

const (
	logicalDiskReadsPerSecPath      = `\LogicalDisk(*)\Disk Reads/sec`
	logicalDiskWritesPerSecPath     = `\LogicalDisk(*)\Disk Writes/sec`
	logicalDiskReadBytesPerSecPath  = `\LogicalDisk(*)\Disk Read Bytes/sec`
	logicalDiskWriteBytesPerSecPath = `\LogicalDisk(*)\Disk Write Bytes/sec`
)

// scraper for Disk Metrics
type scraper struct {
	config *Config

	startTime      pdata.TimestampUnixNano
	prevScrapeTime time.Time

	diskReadBytesPerSecCounter  pdh.PerfCounterScraper
	diskWriteBytesPerSecCounter pdh.PerfCounterScraper
	diskReadsPerSecCounter      pdh.PerfCounterScraper
	diskWritesPerSecCounter     pdh.PerfCounterScraper

	cumulativeDiskIO  cumulativeDiskValues
	cumulativeDiskOps cumulativeDiskValues
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
func newDiskScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{
		config:            cfg,
		cumulativeDiskIO:  map[string]*value{},
		cumulativeDiskOps: map[string]*value{},
	}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	s.startTime = pdata.TimestampUnixNano(uint64(time.Now().UnixNano()))
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

	return componenterror.CombineErrors(errors)
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (pdata.MetricSlice, error) {
	now := time.Now()
	durationSinceLastScraped := now.Sub(s.prevScrapeTime).Seconds()
	s.prevScrapeTime = now

	metrics := pdata.NewMetricSlice()

	var errors []error

	err := s.scrapeAndAppendDiskIOMetric(metrics, durationSinceLastScraped)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendDiskOpsMetric(metrics, durationSinceLastScraped)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

func (s *scraper) scrapeAndAppendDiskIOMetric(metrics pdata.MetricSlice, durationSinceLastScraped float64) error {
	diskReadBytesPerSecValues, err := s.diskReadBytesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	diskWriteBytesPerSecValues, err := s.diskWriteBytesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	for _, diskReadBytesPerSec := range diskReadBytesPerSecValues {
		s.cumulativeDiskIO.getOrAdd(diskReadBytesPerSec.InstanceName).read += (diskReadBytesPerSec.Value * durationSinceLastScraped)
	}

	for _, diskWriteBytesPerSec := range diskWriteBytesPerSecValues {
		s.cumulativeDiskIO.getOrAdd(diskWriteBytesPerSec.InstanceName).write += (diskWriteBytesPerSec.Value * durationSinceLastScraped)
	}

	idx := metrics.Len()
	metrics.Resize(idx + 1)
	initializeDiskIOMetric(metrics.At(idx), s.startTime, s.cumulativeDiskIO)
	return nil
}

func initializeDiskIOMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, io cumulativeDiskValues) {
	diskIODescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(io))

	idx := 0
	for device, value := range io {
		initializeDataPoint(idps.At(idx+0), startTime, device, readDirectionLabelValue, int64(value.read))
		initializeDataPoint(idps.At(idx+1), startTime, device, writeDirectionLabelValue, int64(value.write))
		idx += 2
	}
}

func (s *scraper) scrapeAndAppendDiskOpsMetric(metrics pdata.MetricSlice, durationSinceLastScraped float64) error {
	diskReadsPerSecValues, err := s.diskReadsPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	diskWritesPerSecValues, err := s.diskWritesPerSecCounter.ScrapeData()
	if err != nil {
		return err
	}

	for _, diskReadsPerSec := range diskReadsPerSecValues {
		s.cumulativeDiskOps.getOrAdd(diskReadsPerSec.InstanceName).read += (diskReadsPerSec.Value * durationSinceLastScraped)
	}

	for _, diskWritesPerSec := range diskWritesPerSecValues {
		s.cumulativeDiskOps.getOrAdd(diskWritesPerSec.InstanceName).write += (diskWritesPerSec.Value * durationSinceLastScraped)
	}

	idx := metrics.Len()
	metrics.Resize(idx + 1)
	initializeDiskOpsMetric(metrics.At(idx), s.startTime, s.cumulativeDiskOps)
	return nil
}

func initializeDiskOpsMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, ops cumulativeDiskValues) {
	diskOpsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ops))

	idx := 0
	for device, value := range ops {
		initializeDataPoint(idps.At(idx+0), startTime, device, readDirectionLabelValue, int64(value.read))
		initializeDataPoint(idps.At(idx+1), startTime, device, writeDirectionLabelValue, int64(value.write))
		idx += 2
	}
}

func initializeDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
