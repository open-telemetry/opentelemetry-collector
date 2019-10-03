// Copyright 2019, OpenTelemetry Authors
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

package vmmetricsreceiver

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"contrib.go.opencensus.io/resource/auto"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/prometheus/procfs"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// VMMetricsCollector is a struct that collects and reports VM and process metrics (cpu, mem, etc).
type VMMetricsCollector struct {
	consumer consumer.MetricsConsumer

	startTime time.Time

	fs        procfs.FS
	processFs procfs.FS
	pid       int

	scrapeInterval time.Duration
	metricPrefix   string
	done           chan struct{}
}

const (
	defaultMountPoint     = procfs.DefaultMountPoint // "/proc"
	defaultScrapeInterval = 10 * time.Second
)

var rsc *resourcepb.Resource
var resourceDetectionSync sync.Once

// NewVMMetricsCollector creates a new set of VM and Process Metrics (mem, cpu).
func NewVMMetricsCollector(si time.Duration, mountPoint, processMountPoint, prefix string, consumer consumer.MetricsConsumer) (*VMMetricsCollector, error) {
	if mountPoint == "" {
		mountPoint = defaultMountPoint
	}
	if processMountPoint == "" {
		processMountPoint = defaultMountPoint
	}
	if si <= 0 {
		si = defaultScrapeInterval
	}
	fs, err := procfs.NewFS(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create new VMMetricsCollector: %s", err)
	}
	processFs, err := procfs.NewFS(processMountPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create new VMMetricsCollector: %s", err)
	}
	vmc := &VMMetricsCollector{
		consumer:       consumer,
		startTime:      time.Now(),
		fs:             fs,
		processFs:      processFs,
		pid:            os.Getpid(),
		scrapeInterval: si,
		metricPrefix:   prefix,
		done:           make(chan struct{}),
	}

	return vmc, nil
}

func detectResource() {
	resourceDetectionSync.Do(func() {
		res, err := auto.Detect(context.Background())
		if err != nil {
			panic(fmt.Sprintf("Resource detection failed, err:%v", err))
		}
		if res != nil {
			rsc = &resourcepb.Resource{
				Type:   res.Type,
				Labels: make(map[string]string, len(res.Labels)),
			}
			for k, v := range res.Labels {
				rsc.Labels[k] = v
			}
		}
	})
}

// StartCollection starts a ticker'd goroutine that will scrape and export vm metrics periodically.
func (vmc *VMMetricsCollector) StartCollection() {
	detectResource()

	go func() {
		ticker := time.NewTicker(vmc.scrapeInterval)
		for {
			select {
			case <-ticker.C:
				vmc.scrapeAndExport()

			case <-vmc.done:
				return
			}
		}
	}()
}

// StopCollection stops the collection of metric information
func (vmc *VMMetricsCollector) StopCollection() {
	close(vmc.done)
}

func (vmc *VMMetricsCollector) scrapeAndExport() {
	ctx, span := trace.StartSpan(context.Background(), "VMMetricsCollector.scrapeAndExport")
	defer span.End()

	metrics := make([]*metricspb.Metric, 0, len(vmMetricDescriptors))
	var errs []error

	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)
	metrics = append(
		metrics,
		&metricspb.Metric{
			MetricDescriptor: metricAllocMem,
			Resource:         rsc,
			Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(ms.Alloc)},
		},
		&metricspb.Metric{
			MetricDescriptor: metricTotalAllocMem,
			Resource:         rsc,
			Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(ms.TotalAlloc)},
		},
		&metricspb.Metric{
			MetricDescriptor: metricSysMem,
			Resource:         rsc,
			Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(ms.Sys)},
		},
	)

	var proc procfs.Proc
	var err error
	proc, err = vmc.processFs.Proc(vmc.pid)
	if err == nil {
		procStat, err := proc.Stat()
		if err == nil {
			metrics = append(
				metrics,
				&metricspb.Metric{
					MetricDescriptor: metricProcessCPUSeconds,
					Resource:         rsc,
					Timeseries:       []*metricspb.TimeSeries{vmc.getDoubleTimeSeries(procStat.CPUTime(), nil)},
				},
			)
		}
	}
	if err != nil {
		errs = append(errs, err)
	}

	stat, err := vmc.fs.Stat()
	if err == nil {
		cpuStat := stat.CPUTotal
		metrics = append(
			metrics,
			&metricspb.Metric{
				MetricDescriptor: metricProcessesRunning,
				Resource:         rsc,
				Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(stat.ProcessesRunning)},
			},
			&metricspb.Metric{
				MetricDescriptor: metricProcessesBlocked,
				Resource:         rsc,
				Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(stat.ProcessesBlocked)},
			},
			&metricspb.Metric{
				MetricDescriptor: metricProcessesCreated,
				Resource:         rsc,
				Timeseries:       []*metricspb.TimeSeries{vmc.getInt64TimeSeries(stat.ProcessCreated)},
			},
			&metricspb.Metric{
				MetricDescriptor: metricCPUSeconds,
				Resource:         rsc,
				Timeseries: []*metricspb.TimeSeries{
					vmc.getDoubleTimeSeries(cpuStat.User, labelValueCPUUser),
					vmc.getDoubleTimeSeries(cpuStat.System, labelValueCPUSystem),
					vmc.getDoubleTimeSeries(cpuStat.Idle, labelValueCPUIdle),
					vmc.getDoubleTimeSeries(cpuStat.Nice, labelValueCPUNice),
					vmc.getDoubleTimeSeries(cpuStat.Iowait, labelValueCPUIOWait),
				},
			},
		)
	} else {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping VM metrics: %v", oterr.CombineErrors(errs))})
	}

	if len(metrics) > 0 {
		vmc.consumer.ConsumeMetricsData(ctx, consumerdata.MetricsData{Metrics: metrics})
	}
}

func (vmc *VMMetricsCollector) getInt64TimeSeries(val uint64) *metricspb.TimeSeries {
	return &metricspb.TimeSeries{
		StartTimestamp: internal.TimeToTimestamp(vmc.startTime),
		Points:         []*metricspb.Point{{Timestamp: internal.TimeToTimestamp(time.Now()), Value: &metricspb.Point_Int64Value{Int64Value: int64(val)}}},
	}
}

func (vmc *VMMetricsCollector) getDoubleTimeSeries(val float64, labelVal *metricspb.LabelValue) *metricspb.TimeSeries {
	var labelVals []*metricspb.LabelValue
	if labelVal != nil {
		labelVals = append(labelVals, labelVal)
	}
	return &metricspb.TimeSeries{
		StartTimestamp: internal.TimeToTimestamp(vmc.startTime),
		LabelValues:    labelVals,
		Points:         []*metricspb.Point{{Timestamp: internal.TimeToTimestamp(time.Now()), Value: &metricspb.Point_DoubleValue{DoubleValue: val}}},
	}
}
