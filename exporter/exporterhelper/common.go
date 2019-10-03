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

package exporterhelper

import (
	"go.opencensus.io/trace"
)

var (
	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

// Shutdown specifies the function invoked when the exporter is being shutdown.
type Shutdown func() error

// ExporterOptions contains options concerning how an Exporter is configured.
type ExporterOptions struct {
	// TODO: Retry logic must be in the same place as metrics recording because
	// if a request is retried we should not record metrics otherwise number of
	// spans received + dropped will be different than the number of received spans
	// in the receiver.
	recordMetrics bool
	recordTrace   bool
	shutdown      Shutdown
}

// ExporterOption apply changes to ExporterOptions.
type ExporterOption func(*ExporterOptions)

// WithMetrics makes new Exporter to record metrics for every request.
func WithMetrics(recordMetrics bool) ExporterOption {
	return func(o *ExporterOptions) {
		o.recordMetrics = recordMetrics
	}
}

// WithTracing makes new Exporter to wrap every request with a trace Span.
func WithTracing(recordTrace bool) ExporterOption {
	return func(o *ExporterOptions) {
		o.recordTrace = recordTrace
	}
}

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown Shutdown) ExporterOption {
	return func(o *ExporterOptions) {
		o.shutdown = shutdown
	}
}

// Construct the ExporterOptions from multiple ExporterOption.
func newExporterOptions(options ...ExporterOption) ExporterOptions {
	var opts ExporterOptions
	for _, op := range options {
		op(&opts)
	}
	return opts
}

func errToStatus(err error) trace.Status {
	if err != nil {
		return trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()}
	}
	return okStatus
}
