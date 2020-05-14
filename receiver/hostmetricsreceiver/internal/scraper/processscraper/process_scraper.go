// Copyright 2020, OpenTelemetry Authors
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

package processscraper

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/process"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// scraper for Process Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	getProcessHandles func() ([]processHandle, error)
}

type processMetadata struct {
	pid      int32
	name     string
	handle   processHandle
	username string
	cmdline  string
}

type processTimes struct {
	*processMetadata
	times *cpu.TimesStat
}

type processMemoryInfo struct {
	*processMetadata
	memoryInfo *process.MemoryInfoStat
}

type processIoCounters struct {
	*processMetadata
	ioCounters *process.IOCountersStat
}

// newProcessScraper creates a Process Scraper
func newProcessScraper(cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, getProcessHandles: getProcessHandlesInternal}

	var err error

	if len(cfg.Include.Names) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Names, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process include filters: %v", err)
		}
	}

	if len(cfg.Exclude.Names) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Names, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process exclude filters: %v", err)
		}
	}

	return scraper, nil
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
	_, span := trace.StartSpan(ctx, "processscraper.ScrapeMetrics")
	defer span.End()

	var errors []error
	metrics := pdata.NewMetricSlice()

	processes, err := s.getProcesses()
	if err != nil {
		errors = append(errors, err)
	}

	err = scrapeAndAppendCPUUsageMetric(metrics, s.startTime, processes)
	if err != nil {
		errors = append(errors, err)
	}

	err = scrapeAndAppendMemoryUsageMetric(metrics, s.startTime, processes)
	if err != nil {
		errors = append(errors, err)
	}

	err = scrapeAndAppendDiskBytesMetric(metrics, s.startTime, processes)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

// getProcesses returns a slice of processMetadata, including handles,
// for all currently running processes. If errors occur obtaining
// information for some processes, an error will be returned, but any
// processes that were successfully obtained will still be returned.
func (s *scraper) getProcesses() ([]*processMetadata, error) {
	handles, err := s.getProcessHandles()
	if err != nil {
		return nil, err
	}

	var errors []error
	metadata := make([]*processMetadata, 0, len(handles))
	for _, handle := range handles {
		name, err := getProcessName(handle)
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading process name for pid %v: %v", handle.GetPid(), err))
			continue
		}

		// filter processes by name
		if (s.includeFS != nil && !s.includeFS.Matches(name)) ||
			(s.excludeFS != nil && s.excludeFS.Matches(name)) {
			continue
		}

		md := &processMetadata{}
		err = initializeProcessMetadata(md, handle, name)
		if err != nil {
			errors = append(errors, err)
		}

		metadata = append(metadata, md)
	}

	if len(errors) > 0 {
		return metadata, componenterror.CombineErrors(errors)
	}

	return metadata, nil
}

func getProcessName(proc processHandle) (string, error) {
	if runtime.GOOS != "windows" {
		return proc.Name()
	}

	// calling proc.Name() is currently prohibitively expensive on Windows, so use the exe name instead (mirrors psutil)
	exe, err := proc.Exe()
	if err != nil {
		return "", err
	}

	return filepath.Base(exe), nil
}

func initializeProcessMetadata(metadata *processMetadata, handle processHandle, name string) error {
	metadata.pid = handle.GetPid()
	metadata.name = name
	metadata.handle = handle

	var errors []error
	var err error

	metadata.username, err = handle.Username()
	if err != nil {
		errors = append(errors, fmt.Errorf("error reading process username for process %q (pid %v): %v", metadata.name, metadata.pid, err))
		metadata.username = ""
	}

	metadata.cmdline, err = handle.Cmdline()
	if err != nil {
		errors = append(errors, fmt.Errorf("error reading process cmdline for process %q (pid %v): %v", metadata.name, metadata.pid, err))
		metadata.cmdline = ""
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func scrapeAndAppendCPUUsageMetric(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano, processes []*processMetadata) error {
	cpuTimes := make([]*processTimes, 0, len(processes))

	var errors []error
	for _, metadata := range processes {
		times, err := metadata.handle.Times()
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading process cpu times for process %q (pid %v): %v", metadata.name, metadata.pid, err))
		} else {
			cpuTimes = append(cpuTimes, &processTimes{processMetadata: metadata, times: times})
		}
	}

	if len(cpuTimes) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + 1)
		initializeCPUUsageMetric(metrics.At(startIdx), startTime, cpuTimes)
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func initializeCPUUsageMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []*processTimes) {
	metricCPUUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, times := range cpuTimes {
		appendCPUStateTimes(idps, i*cpuStatesLen, startTime, times.processMetadata, times.times)
	}
}

func scrapeAndAppendMemoryUsageMetric(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano, processes []*processMetadata) error {
	memoryInfos := make([]*processMemoryInfo, 0, len(processes))

	var errors []error
	for _, metadata := range processes {
		mem, err := metadata.handle.MemoryInfo()
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading process memory info for process %q (pid %v): %v", metadata.name, metadata.pid, err))
		} else {
			memoryInfos = append(memoryInfos, &processMemoryInfo{processMetadata: metadata, memoryInfo: mem})
		}
	}

	if len(memoryInfos) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + 1)
		initializeMemoryUsageMetric(metrics.At(startIdx), startTime, memoryInfos)
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func initializeMemoryUsageMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, memoryInfo []*processMemoryInfo) {
	metricMemoryUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(memoryInfo))
	for i, mem := range memoryInfo {
		initializeProcessDataPoint(idps.At(i), startTime, mem.processMetadata, int64(mem.memoryInfo.RSS))
	}
}

func scrapeAndAppendDiskBytesMetric(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano, processes []*processMetadata) error {
	ioCounters := make([]*processIoCounters, 0, len(processes))

	var errors []error
	for _, metadata := range processes {
		io, err := metadata.handle.IOCounters()
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading process disk usage for process %q (pid %v): %v", metadata.name, metadata.pid, err))
		} else {
			ioCounters = append(ioCounters, &processIoCounters{processMetadata: metadata, ioCounters: io})
		}
	}

	if len(ioCounters) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + 1)
		initializeDiskBytesMetric(metrics.At(startIdx), startTime, ioCounters)
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func initializeDiskBytesMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, ioCounters []*processIoCounters) {
	metricDiskBytesDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for _, io := range ioCounters {
		initializeProcessDataPoint(idps.At(idx+0), startTime, io.processMetadata, int64(io.ioCounters.ReadBytes), directionLabelName, readDirectionLabelValue)
		initializeProcessDataPoint(idps.At(idx+1), startTime, io.processMetadata, int64(io.ioCounters.WriteBytes), directionLabelName, writeDirectionLabelValue)
		idx += 2
	}
}

func initializeProcessDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, metadata *processMetadata, value int64, additionalLabels ...string) {
	labelsMap := dataPoint.LabelsMap()

	labelsMap.Insert(processLabelName, metadata.name)
	if metadata.username != "" {
		labelsMap.Insert(usernameLabelName, metadata.username)
	}
	if metadata.cmdline != "" {
		labelsMap.Insert(cmdlineLabelName, metadata.cmdline)
	}

	for i := 0; i < len(additionalLabels); i += 2 {
		labelsMap.Insert(additionalLabels[i], additionalLabels[i+1])
	}

	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
