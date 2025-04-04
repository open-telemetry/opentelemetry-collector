// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
)

// LogsBuilder provides an interface for scrapers to report logs while taking care of all the transformations
// required to produce log representation defined in metadata and user config.
type LogsBuilder struct {
	logsBuffer       plog.Logs
	logRecordsBuffer plog.LogRecordSlice
	buildInfo        component.BuildInfo // contains version information.
}

// LogBuilderOption applies changes to default logs builder.
type LogBuilderOption interface {
	apply(*LogsBuilder)
}

func NewLogsBuilder(settings receiver.Settings) *LogsBuilder {
	lb := &LogsBuilder{
		logsBuffer:       plog.NewLogs(),
		logRecordsBuffer: plog.NewLogRecordSlice(),
		buildInfo:        settings.BuildInfo,
	}

	return lb
}

// NewResourceBuilder returns a new resource builder that should be used to build a resource associated with for the emitted logs.
func (lb *LogsBuilder) NewResourceBuilder() *ResourceBuilder {
	return NewResourceBuilder(ResourceAttributesConfig{})
}

// ResourceLogsOption applies changes to provided resource logs.
type ResourceLogsOption interface {
	apply(plog.ResourceLogs)
}

type resourceLogsOptionFunc func(plog.ResourceLogs)

func (rlof resourceLogsOptionFunc) apply(rl plog.ResourceLogs) {
	rlof(rl)
}

// WithLogsResource sets the provided resource on the emitted ResourceLogs.
// It's recommended to use ResourceBuilder to create the resource.
func WithLogsResource(res pcommon.Resource) ResourceLogsOption {
	return resourceLogsOptionFunc(func(rl plog.ResourceLogs) {
		res.CopyTo(rl.Resource())
	})
}

// AppendLogRecord adds a log record to the logs builder.
func (lb *LogsBuilder) AppendLogRecord(lr plog.LogRecord) {
	lr.MoveTo(lb.logRecordsBuffer.AppendEmpty())
}

// EmitForResource saves all the generated logs under a new resource and updates the internal state to be ready for
// recording another set of log records as part of another resource. This function can be helpful when one scraper
// needs to emit logs from several resources. Otherwise calling this function is not required,
// just `Emit` function can be called instead.
// Resource attributes should be provided as ResourceLogsOption arguments.
func (lb *LogsBuilder) EmitForResource(options ...ResourceLogsOption) {
	rl := lb.logsBuffer.ResourceLogs().AppendEmpty()
	rl.SetSchemaUrl(conventions.SchemaURL)
	ils := rl.ScopeLogs().AppendEmpty()
	ils.Scope().SetName(ScopeName)
	ils.Scope().SetVersion(lb.buildInfo.Version)

	for _, op := range options {
		op.apply(rl)
	}

	if lb.logRecordsBuffer.Len() > 0 {
		lb.logRecordsBuffer.MoveAndAppendTo(ils.LogRecords())
		lb.logRecordsBuffer = plog.NewLogRecordSlice()
	}
}

// Emit returns all the logs accumulated by the logs builder and updates the internal state to be ready for
// recording another set of logs. This function will be responsible for applying all the transformations required to
// produce logs representation defined in metadata and user config.
func (lb *LogsBuilder) Emit(options ...ResourceLogsOption) plog.Logs {
	lb.EmitForResource(options...)
	logs := lb.logsBuffer
	lb.logsBuffer = plog.NewLogs()
	return logs
}
