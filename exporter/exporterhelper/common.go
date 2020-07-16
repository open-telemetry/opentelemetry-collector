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
	"sync"
	"time"

	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

var (
	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

type TimeoutSettings struct {
	// Timeout is the timeout for each operation.
	Timeout time.Duration `mapstructure:"timeout"`
}

func CreateDefaultTimeoutSettings() TimeoutSettings {
	return TimeoutSettings{
		Timeout: 5 * time.Second,
	}
}

type settings struct {
	configmodels.Exporter
	TimeoutSettings
	QueuedSettings
	RetrySettings
}

type request interface {
	context() context.Context
	setContext(context.Context)
	export(ctx context.Context) (int, error)
	// Returns a new queue request that contains the items left to be exported.
	onPartialError(consumererror.PartialError) request
	// Returns the cnt of spans/metric points or log records.
	count() int
}

type requestSender interface {
	send(req request) (int, error)
}

type baseRequest struct {
	ctx context.Context
}

func (req *baseRequest) context() context.Context {
	return req.ctx
}

func (req *baseRequest) setContext(ctx context.Context) {
	req.ctx = ctx
}

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

// WithShutdown overrides the default TimeoutSettings for an exporter.
// The default TimeoutSettings is 5 seconds.
func WithTimeout(timeout TimeoutSettings) ExporterOption {
	return func(o *baseExporter) {
		o.cfg.TimeoutSettings = timeout
	}
}

// WithRetry overrides the default RetrySettings for an exporter.
// The default RetrySettings is to disable retries.
func WithRetry(retry RetrySettings) ExporterOption {
	return func(o *baseExporter) {
		o.cfg.RetrySettings = retry
	}
}

// WithQueued overrides the default QueuedSettings for an exporter.
// The default QueuedSettings is to disable queueing.
func WithQueued(queued QueuedSettings) ExporterOption {
	return func(o *baseExporter) {
		o.cfg.QueuedSettings = queued
	}
}

// internalOptions contains internalOptions concerning how an Exporter is configured.
type baseExporter struct {
	cfg          *settings
	sender       requestSender
	rSender      *retrySender
	qSender      *queuedSender
	start        Start
	shutdown     Shutdown
	startOnce    sync.Once
	shutdownOnce sync.Once
}

// Construct the internalOptions from multiple ExporterOption.
func newBaseExporter(cfg configmodels.Exporter, options ...ExporterOption) *baseExporter {
	be := &baseExporter{
		cfg: &settings{
			Exporter:        cfg,
			TimeoutSettings: CreateDefaultTimeoutSettings(),
			// TODO: Enable queuing by default (call CreateDefaultQueuedSettings
			QueuedSettings: QueuedSettings{Disabled: true},
			// TODO: Enable retry by default (call CreateDefaultRetrySettings)
			RetrySettings: RetrySettings{Disabled: true},
		},
	}

	for _, op := range options {
		op(be)
	}

	if be.start == nil {
		be.start = func(ctx context.Context, host component.Host) error { return nil }
	}

	if be.shutdown == nil {
		be.shutdown = func(ctx context.Context) error { return nil }
	}

	be.sender = &timeoutSender{cfg: &be.cfg.TimeoutSettings}

	be.rSender = newRetrySender(&be.cfg.RetrySettings, be.sender)
	be.sender = be.rSender

	be.qSender = newQueuedSender(&be.cfg.QueuedSettings, be.sender)
	be.sender = be.qSender

	return be
}

func (be *baseExporter) Start(ctx context.Context, host component.Host) error {
	err := componenterror.ErrAlreadyStarted
	be.startOnce.Do(func() {
		// First start the nextSender
		err = be.start(ctx, host)
		if err != nil {
			return
		}

		// If no error then start the queuedSender
		be.qSender.start()
	})
	return err
}

// Shutdown stops the nextSender and is invoked during shutdown.
func (be *baseExporter) Shutdown(ctx context.Context) error {
	err := componenterror.ErrAlreadyStopped
	be.shutdownOnce.Do(func() {
		// First stop the retry goroutines
		be.rSender.shutdown()

		// All operations will try to export once but will not retry because retrying was disabled when be.rSender stopped.
		be.qSender.shutdown()

		// Last shutdown the nextSender itself.
		err = be.shutdown(ctx)
	})
	return err
}

type timeoutSender struct {
	cfg *TimeoutSettings
}

func (te *timeoutSender) send(req request) (int, error) {
	// Intentionally don't overwrite the context inside the request, because in case of retries deadline will not be
	// updated because this deadline most likely is before the next one.
	ctx := req.context()
	if te.cfg.Timeout > 0 {
		var cancelFunc func()
		ctx, cancelFunc = context.WithTimeout(req.context(), te.cfg.Timeout)
		defer cancelFunc()
	}
	return req.export(ctx)
}
