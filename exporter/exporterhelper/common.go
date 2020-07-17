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

package exporterhelper

import (
	"context"

	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component"
)

var (
	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

// Start specifies the function invoked when the exporter is being started.
type Start func(context.Context, component.Host) error

// Shutdown specifies the function invoked when the exporter is being shutdown.
type Shutdown func(context.Context) error

// ExporterOption apply changes to internalOptions.
type ExporterOption func(*baseExporter)

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown Shutdown) ExporterOption {
	return func(o *baseExporter) {
		o.shutdown = shutdown
	}
}

// WithStart overrides the default Start function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithStart(start Start) ExporterOption {
	return func(o *baseExporter) {
		o.start = start
	}
}

// internalOptions contains internalOptions concerning how an Exporter is configured.
type baseExporter struct {
	exporterFullName string
	start            Start
	shutdown         Shutdown
}

// Construct the internalOptions from multiple ExporterOption.
func newBaseExporter(exporterFullName string, options ...ExporterOption) baseExporter {
	be := baseExporter{
		exporterFullName: exporterFullName,
	}

	for _, op := range options {
		op(&be)
	}

	return be
}

func (be *baseExporter) Start(ctx context.Context, host component.Host) error {
	if be.start != nil {
		return be.start(ctx, host)
	}
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (be *baseExporter) Shutdown(ctx context.Context) error {
	if be.shutdown != nil {
		return be.shutdown(ctx)
	}
	return nil
}
