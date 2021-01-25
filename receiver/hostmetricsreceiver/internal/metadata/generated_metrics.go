// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Type is the component type name.
const Type configmodels.Type = "hostmetricsreceiver"

// MetricIntf is an interface to generically interact with generated metric.
type MetricIntf interface {
	Name() string
	New() pdata.Metric
	Init(metric pdata.Metric)
}

// Intentionally not exposing this so that it is opaque and can change freely.
type metricImpl struct {
	name     string
	initFunc func(pdata.Metric)
}

// Name returns the metric name.
func (m *metricImpl) Name() string {
	return m.name
}

// New creates a metric object preinitialized.
func (m *metricImpl) New() pdata.Metric {
	metric := pdata.NewMetric()
	m.Init(metric)
	return metric
}

// Init initializes the provided metric object.
func (m *metricImpl) Init(metric pdata.Metric) {
	m.initFunc(metric)
}

type metricStruct struct {
	SystemCPULoadAverage15m     MetricIntf
	SystemCPULoadAverage1m      MetricIntf
	SystemCPULoadAverage5m      MetricIntf
	SystemCPUTime               MetricIntf
	SystemDiskIo                MetricIntf
	SystemDiskIoTime            MetricIntf
	SystemDiskMerged            MetricIntf
	SystemDiskOperationTime     MetricIntf
	SystemDiskOperations        MetricIntf
	SystemDiskPendingOperations MetricIntf
	SystemDiskWeightedIoTime    MetricIntf
	SystemFilesystemInodesUsage MetricIntf
	SystemFilesystemUsage       MetricIntf
	SystemMemoryUsage           MetricIntf
}

// Names returns a list of all the metric name strings.
func (m *metricStruct) Names() []string {
	return []string{
		"system.cpu.load_average.15m",
		"system.cpu.load_average.1m",
		"system.cpu.load_average.5m",
		"system.cpu.time",
		"system.disk.io",
		"system.disk.io_time",
		"system.disk.merged",
		"system.disk.operation_time",
		"system.disk.operations",
		"system.disk.pending_operations",
		"system.disk.weighted_io_time",
		"system.filesystem.inodes.usage",
		"system.filesystem.usage",
		"system.memory.usage",
	}
}

var metricsByName = map[string]MetricIntf{
	"system.cpu.load_average.15m":    Metrics.SystemCPULoadAverage15m,
	"system.cpu.load_average.1m":     Metrics.SystemCPULoadAverage1m,
	"system.cpu.load_average.5m":     Metrics.SystemCPULoadAverage5m,
	"system.cpu.time":                Metrics.SystemCPUTime,
	"system.disk.io":                 Metrics.SystemDiskIo,
	"system.disk.io_time":            Metrics.SystemDiskIoTime,
	"system.disk.merged":             Metrics.SystemDiskMerged,
	"system.disk.operation_time":     Metrics.SystemDiskOperationTime,
	"system.disk.operations":         Metrics.SystemDiskOperations,
	"system.disk.pending_operations": Metrics.SystemDiskPendingOperations,
	"system.disk.weighted_io_time":   Metrics.SystemDiskWeightedIoTime,
	"system.filesystem.inodes.usage": Metrics.SystemFilesystemInodesUsage,
	"system.filesystem.usage":        Metrics.SystemFilesystemUsage,
	"system.memory.usage":            Metrics.SystemMemoryUsage,
}

func (m *metricStruct) ByName(n string) MetricIntf {
	return metricsByName[n]
}

func (m *metricStruct) FactoriesByName() map[string]func() pdata.Metric {
	return map[string]func() pdata.Metric{
		Metrics.SystemCPULoadAverage15m.Name():     Metrics.SystemCPULoadAverage15m.New,
		Metrics.SystemCPULoadAverage1m.Name():      Metrics.SystemCPULoadAverage1m.New,
		Metrics.SystemCPULoadAverage5m.Name():      Metrics.SystemCPULoadAverage5m.New,
		Metrics.SystemCPUTime.Name():               Metrics.SystemCPUTime.New,
		Metrics.SystemDiskIo.Name():                Metrics.SystemDiskIo.New,
		Metrics.SystemDiskIoTime.Name():            Metrics.SystemDiskIoTime.New,
		Metrics.SystemDiskMerged.Name():            Metrics.SystemDiskMerged.New,
		Metrics.SystemDiskOperationTime.Name():     Metrics.SystemDiskOperationTime.New,
		Metrics.SystemDiskOperations.Name():        Metrics.SystemDiskOperations.New,
		Metrics.SystemDiskPendingOperations.Name(): Metrics.SystemDiskPendingOperations.New,
		Metrics.SystemDiskWeightedIoTime.Name():    Metrics.SystemDiskWeightedIoTime.New,
		Metrics.SystemFilesystemInodesUsage.Name(): Metrics.SystemFilesystemInodesUsage.New,
		Metrics.SystemFilesystemUsage.Name():       Metrics.SystemFilesystemUsage.New,
		Metrics.SystemMemoryUsage.Name():           Metrics.SystemMemoryUsage.New,
	}
}

// Metrics contains a set of methods for each metric that help with
// manipulating those metrics.
var Metrics = &metricStruct{
	&metricImpl{
		"system.cpu.load_average.15m",
		func(metric pdata.Metric) {
			metric.SetName("system.cpu.load_average.15m")
			metric.SetDescription("Average CPU Load over 15 minutes.")
			metric.SetUnit("1")
			metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
		},
	},
	&metricImpl{
		"system.cpu.load_average.1m",
		func(metric pdata.Metric) {
			metric.SetName("system.cpu.load_average.1m")
			metric.SetDescription("Average CPU Load over 1 minute.")
			metric.SetUnit("1")
			metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
		},
	},
	&metricImpl{
		"system.cpu.load_average.5m",
		func(metric pdata.Metric) {
			metric.SetName("system.cpu.load_average.5m")
			metric.SetDescription("Average CPU Load over 5 minutes.")
			metric.SetUnit("1")
			metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
		},
	},
	&metricImpl{
		"system.cpu.time",
		func(metric pdata.Metric) {
			metric.SetName("system.cpu.time")
			metric.SetDescription("Total CPU seconds broken down by different states.")
			metric.SetUnit("s")
			metric.SetDataType(pdata.MetricDataTypeDoubleSum)
			metric.DoubleSum().SetIsMonotonic(true)
			metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.io",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.io")
			metric.SetDescription("Disk bytes transferred.")
			metric.SetUnit("By")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(true)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.io_time",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.io_time")
			metric.SetDescription("Time disk spent activated. On Windows, this is calculated as the inverse of disk idle time.")
			metric.SetUnit("s")
			metric.SetDataType(pdata.MetricDataTypeDoubleSum)
			metric.DoubleSum().SetIsMonotonic(true)
			metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.merged",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.merged")
			metric.SetDescription("The number of disk reads merged into single physical disk access operations.")
			metric.SetUnit("{operations}")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(true)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.operation_time",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.operation_time")
			metric.SetDescription("Time spent in disk operations.")
			metric.SetUnit("s")
			metric.SetDataType(pdata.MetricDataTypeDoubleSum)
			metric.DoubleSum().SetIsMonotonic(true)
			metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.operations",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.operations")
			metric.SetDescription("Disk operations count.")
			metric.SetUnit("{operations}")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(true)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.pending_operations",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.pending_operations")
			metric.SetDescription("The queue size of pending I/O operations.")
			metric.SetUnit("{operations}")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(false)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.disk.weighted_io_time",
		func(metric pdata.Metric) {
			metric.SetName("system.disk.weighted_io_time")
			metric.SetDescription("Time disk spent activated multiplied by the queue length.")
			metric.SetUnit("s")
			metric.SetDataType(pdata.MetricDataTypeDoubleSum)
			metric.DoubleSum().SetIsMonotonic(true)
			metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.filesystem.inodes.usage",
		func(metric pdata.Metric) {
			metric.SetName("system.filesystem.inodes.usage")
			metric.SetDescription("FileSystem inodes used.")
			metric.SetUnit("{inodes}")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(false)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.filesystem.usage",
		func(metric pdata.Metric) {
			metric.SetName("system.filesystem.usage")
			metric.SetDescription("Filesystem bytes used.")
			metric.SetUnit("By")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(false)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
	&metricImpl{
		"system.memory.usage",
		func(metric pdata.Metric) {
			metric.SetName("system.memory.usage")
			metric.SetDescription("Bytes of memory in use.")
			metric.SetUnit("By")
			metric.SetDataType(pdata.MetricDataTypeIntSum)
			metric.IntSum().SetIsMonotonic(false)
			metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		},
	},
}

// M contains a set of methods for each metric that help with
// manipulating those metrics. M is an alias for Metrics
var M = Metrics

// Labels contains the possible metric labels that can be used.
var Labels = struct {
	// Cpu (CPU number starting at 0.)
	Cpu string
	// CPUState (Breakdown of CPU usage by type.)
	CPUState string
	// DiskDevice (Name of the disk.)
	DiskDevice string
	// DiskDirection (Direction of flow of bytes/opertations (read or write).)
	DiskDirection string
	// FilesystemDevice (Identifier of the filesystem.)
	FilesystemDevice string
	// FilesystemMode (Mountpoint mode such "ro", "rw", etc.)
	FilesystemMode string
	// FilesystemMountpoint (Mountpoint path.)
	FilesystemMountpoint string
	// FilesystemState (Breakdown of filesystem usage by type.)
	FilesystemState string
	// FilesystemType (Filesystem type, such as, "ext4", "tmpfs", etc.)
	FilesystemType string
	// MemState (Breakdown of memory usage by type.)
	MemState string
}{
	"cpu",
	"state",
	"device",
	"direction",
	"device",
	"mode",
	"mountpoint",
	"state",
	"type",
	"state",
}

// L contains the possible metric labels that can be used. L is an alias for
// Labels.
var L = Labels

// LabelCPUState are the possible values that the label "cpu.state" can have.
var LabelCPUState = struct {
	Idle      string
	Interrupt string
	Nice      string
	Softirq   string
	Steal     string
	System    string
	User      string
	Wait      string
}{
	"idle",
	"interrupt",
	"nice",
	"softirq",
	"steal",
	"system",
	"user",
	"wait",
}

// LabelDiskDirection are the possible values that the label "disk.direction" can have.
var LabelDiskDirection = struct {
	Read  string
	Write string
}{
	"read",
	"write",
}

// LabelFilesystemState are the possible values that the label "filesystem.state" can have.
var LabelFilesystemState = struct {
	Free     string
	Reserved string
	Used     string
}{
	"free",
	"reserved",
	"used",
}

// LabelMemState are the possible values that the label "mem.state" can have.
var LabelMemState = struct {
	Buffered          string
	Cached            string
	Inactive          string
	Free              string
	SlabReclaimable   string
	SlabUnreclaimable string
	Used              string
}{
	"buffered",
	"cached",
	"inactive",
	"free",
	"slab_reclaimable",
	"slab_unreclaimable",
	"used",
}
