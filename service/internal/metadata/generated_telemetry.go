// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("go.opentelemetry.io/collector/service")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("go.opentelemetry.io/collector/service")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                             metric.Meter
	mu                                sync.Mutex
	registrations                     []metric.Registration
	ProcessCPUSeconds                 metric.Float64ObservableCounter
	ProcessMemoryRss                  metric.Int64ObservableGauge
	ProcessRuntimeHeapAllocBytes      metric.Int64ObservableGauge
	ProcessRuntimeTotalAllocBytes     metric.Int64ObservableCounter
	ProcessRuntimeTotalSysMemoryBytes metric.Int64ObservableGauge
	ProcessUptime                     metric.Float64ObservableCounter
}

// TelemetryBuilderOption applies changes to default builder.
type TelemetryBuilderOption interface {
	apply(*TelemetryBuilder)
}

type telemetryBuilderOptionFunc func(mb *TelemetryBuilder)

func (tbof telemetryBuilderOptionFunc) apply(mb *TelemetryBuilder) {
	tbof(mb)
}

// RegisterProcessCPUSecondsCallback sets callback for observable ProcessCPUSeconds metric.
func (builder *TelemetryBuilder) RegisterProcessCPUSecondsCallback(cb metric.Float64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerFloat64{inst: builder.ProcessCPUSeconds, obs: o})
		return nil
	}, builder.ProcessCPUSeconds)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

// RegisterProcessMemoryRssCallback sets callback for observable ProcessMemoryRss metric.
func (builder *TelemetryBuilder) RegisterProcessMemoryRssCallback(cb metric.Int64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerInt64{inst: builder.ProcessMemoryRss, obs: o})
		return nil
	}, builder.ProcessMemoryRss)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

// RegisterProcessRuntimeHeapAllocBytesCallback sets callback for observable ProcessRuntimeHeapAllocBytes metric.
func (builder *TelemetryBuilder) RegisterProcessRuntimeHeapAllocBytesCallback(cb metric.Int64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerInt64{inst: builder.ProcessRuntimeHeapAllocBytes, obs: o})
		return nil
	}, builder.ProcessRuntimeHeapAllocBytes)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

// RegisterProcessRuntimeTotalAllocBytesCallback sets callback for observable ProcessRuntimeTotalAllocBytes metric.
func (builder *TelemetryBuilder) RegisterProcessRuntimeTotalAllocBytesCallback(cb metric.Int64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerInt64{inst: builder.ProcessRuntimeTotalAllocBytes, obs: o})
		return nil
	}, builder.ProcessRuntimeTotalAllocBytes)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

// RegisterProcessRuntimeTotalSysMemoryBytesCallback sets callback for observable ProcessRuntimeTotalSysMemoryBytes metric.
func (builder *TelemetryBuilder) RegisterProcessRuntimeTotalSysMemoryBytesCallback(cb metric.Int64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerInt64{inst: builder.ProcessRuntimeTotalSysMemoryBytes, obs: o})
		return nil
	}, builder.ProcessRuntimeTotalSysMemoryBytes)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

// RegisterProcessUptimeCallback sets callback for observable ProcessUptime metric.
func (builder *TelemetryBuilder) RegisterProcessUptimeCallback(cb metric.Float64Callback) error {
	reg, err := builder.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cb(ctx, &observerFloat64{inst: builder.ProcessUptime, obs: o})
		return nil
	}, builder.ProcessUptime)
	if err != nil {
		return err
	}
	builder.mu.Lock()
	defer builder.mu.Unlock()
	builder.registrations = append(builder.registrations, reg)
	return nil
}

type observerInt64 struct {
	embedded.Int64Observer
	inst metric.Int64Observable
	obs  metric.Observer
}

func (oi *observerInt64) Observe(value int64, opts ...metric.ObserveOption) {
	oi.obs.ObserveInt64(oi.inst, value, opts...)
}

type observerFloat64 struct {
	embedded.Float64Observer
	inst metric.Float64Observable
	obs  metric.Observer
}

func (oi *observerFloat64) Observe(value float64, opts ...metric.ObserveOption) {
	oi.obs.ObserveFloat64(oi.inst, value, opts...)
}

// Shutdown unregister all registered callbacks for async instruments.
func (builder *TelemetryBuilder) Shutdown() {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	for _, reg := range builder.registrations {
		reg.Unregister()
	}
}

// NewTelemetryBuilder provides a struct with methods to update all internal telemetry
// for a component
func NewTelemetryBuilder(settings component.TelemetrySettings, options ...TelemetryBuilderOption) (*TelemetryBuilder, error) {
	builder := TelemetryBuilder{}
	for _, op := range options {
		op.apply(&builder)
	}
	builder.meter = Meter(settings)
	var err, errs error
	builder.ProcessCPUSeconds, err = builder.meter.Float64ObservableCounter(
		"otelcol_process_cpu_seconds",
		metric.WithDescription("Total CPU user and system time in seconds [alpha]"),
		metric.WithUnit("s"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessMemoryRss, err = builder.meter.Int64ObservableGauge(
		"otelcol_process_memory_rss",
		metric.WithDescription("Total physical memory (resident set size) [alpha]"),
		metric.WithUnit("By"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessRuntimeHeapAllocBytes, err = builder.meter.Int64ObservableGauge(
		"otelcol_process_runtime_heap_alloc_bytes",
		metric.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc') [alpha]"),
		metric.WithUnit("By"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessRuntimeTotalAllocBytes, err = builder.meter.Int64ObservableCounter(
		"otelcol_process_runtime_total_alloc_bytes",
		metric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc') [alpha]"),
		metric.WithUnit("By"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessRuntimeTotalSysMemoryBytes, err = builder.meter.Int64ObservableGauge(
		"otelcol_process_runtime_total_sys_memory_bytes",
		metric.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys') [alpha]"),
		metric.WithUnit("By"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessUptime, err = builder.meter.Float64ObservableCounter(
		"otelcol_process_uptime",
		metric.WithDescription("Uptime of the process [alpha]"),
		metric.WithUnit("s"),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}
