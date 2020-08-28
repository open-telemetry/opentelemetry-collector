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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/process"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// scraper for Process Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	// for mocking
	bootTime          func() (uint64, error)
	getProcessHandles func() (processHandles, error)
}

// newProcessScraper creates a Process Scraper
func newProcessScraper(cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, bootTime: host.BootTime, getProcessHandles: getProcessHandlesInternal}

	var err error

	if len(cfg.Include.Names) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Names, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Names) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Names, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process exclude filters: %w", err)
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
func (s *scraper) ScrapeMetrics(_ context.Context) (dataold.ResourceMetricsSlice, error) {
	var errs []error

	metadata, err := s.getProcessMetadata()
	if err != nil {
		errs = append(errs, err)
	}

	rms := dataold.NewResourceMetricsSlice()
	rms.Resize(len(metadata))
	for i, md := range metadata {
		rm := rms.At(i)
		md.initializeResource(rm.Resource())

		ilms := rm.InstrumentationLibraryMetrics()
		ilms.Resize(1)
		metrics := ilms.At(0).Metrics()

		now := internal.TimeToUnixNano(time.Now())

		if err = scrapeAndAppendCPUTimeMetric(metrics, s.startTime, now, md.handle); err != nil {
			errs = append(errs, fmt.Errorf("error reading cpu times for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = scrapeAndAppendMemoryUsageMetrics(metrics, now, md.handle); err != nil {
			errs = append(errs, fmt.Errorf("error reading memory info for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = scrapeAndAppendDiskIOMetric(metrics, s.startTime, now, md.handle); err != nil {
			errs = append(errs, fmt.Errorf("error reading disk usage for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}
	}

	return rms, componenterror.CombineErrors(errs)
}

// getProcessMetadata returns a slice of processMetadata, including handles,
// for all currently running processes. If errors occur obtaining information
// for some processes, an error will be returned, but any processes that were
// successfully obtained will still be returned.
func (s *scraper) getProcessMetadata() ([]*processMetadata, error) {
	handles, err := s.getProcessHandles()
	if err != nil {
		return nil, err
	}

	var errs []error
	metadata := make([]*processMetadata, 0, handles.Len())
	for i := 0; i < handles.Len(); i++ {
		pid := handles.Pid(i)
		handle := handles.At(i)

		executable, err := getProcessExecutable(handle)
		if err != nil {
			errs = append(errs, fmt.Errorf("error reading process name for pid %v: %w", pid, err))
			continue
		}

		// filter processes by name
		if (s.includeFS != nil && !s.includeFS.Matches(executable.name)) ||
			(s.excludeFS != nil && s.excludeFS.Matches(executable.name)) {
			continue
		}

		command, err := getProcessCommand(handle)
		if err != nil {
			errs = append(errs, fmt.Errorf("error reading command for process %q (pid %v): %w", executable.name, pid, err))
		}

		username, err := handle.Username()
		if err != nil {
			errs = append(errs, fmt.Errorf("error reading username for process %q (pid %v): %w", executable.name, pid, err))
		}

		md := &processMetadata{
			pid:        pid,
			executable: executable,
			command:    command,
			username:   username,
			handle:     handle,
		}

		metadata = append(metadata, md)
	}

	return metadata, componenterror.CombineErrors(errs)
}

func scrapeAndAppendCPUTimeMetric(metrics dataold.MetricSlice, startTime, now pdata.TimestampUnixNano, handle processHandle) error {
	times, err := handle.Times()
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeCPUTimeMetric(metrics.At(startIdx), startTime, now, times)
	return nil
}

func initializeCPUTimeMetric(metric dataold.Metric, startTime, now pdata.TimestampUnixNano, times *cpu.TimesStat) {
	cpuTimeDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(cpuStatesLen)
	appendCPUTimeStateDataPoints(ddps, startTime, now, times)
}

func scrapeAndAppendMemoryUsageMetrics(metrics dataold.MetricSlice, now pdata.TimestampUnixNano, handle processHandle) error {
	mem, err := handle.MemoryInfo()
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 2)
	initializeMemoryUsageMetric(metrics.At(startIdx+0), physicalMemoryUsageDescriptor, now, int64(mem.RSS))
	initializeMemoryUsageMetric(metrics.At(startIdx+1), virtualMemoryUsageDescriptor, now, int64(mem.VMS))
	return nil
}

func initializeMemoryUsageMetric(metric dataold.Metric, descriptor dataold.MetricDescriptor, now pdata.TimestampUnixNano, usage int64) {
	descriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(1)
	initializeMemoryUsageDataPoint(idps.At(0), now, usage)
}

func initializeMemoryUsageDataPoint(dataPoint dataold.Int64DataPoint, now pdata.TimestampUnixNano, usage int64) {
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(usage)
}

func scrapeAndAppendDiskIOMetric(metrics dataold.MetricSlice, startTime, now pdata.TimestampUnixNano, handle processHandle) error {
	io, err := handle.IOCounters()
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeDiskIOMetric(metrics.At(startIdx), startTime, now, io)
	return nil
}

func initializeDiskIOMetric(metric dataold.Metric, startTime, now pdata.TimestampUnixNano, io *process.IOCountersStat) {
	diskIODescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2)
	initializeDiskIODataPoint(idps.At(0), startTime, now, int64(io.ReadBytes), readDirectionLabelValue)
	initializeDiskIODataPoint(idps.At(1), startTime, now, int64(io.WriteBytes), writeDirectionLabelValue)
}

func initializeDiskIODataPoint(dataPoint dataold.Int64DataPoint, startTime, now pdata.TimestampUnixNano, value int64, directionLabel string) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
