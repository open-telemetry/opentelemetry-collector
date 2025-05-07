// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package extensioncapabilities provides interfaces that can be implemented by extensions
// to provide additional capabilities.
package extensioncapabilities // import "go.opentelemetry.io/collector/extension/extensioncapabilities"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
)

// Dependent is an optional interface that can be implemented by extensions
// that depend on other extensions and must be started only after their dependencies.
// See https://github.com/open-telemetry/opentelemetry-collector/pull/8768 for examples.
type Dependent interface {
	extension.Extension
	Dependencies() []component.ID
}

// PipelineWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the Extension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (note that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the Extension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the Extension can take any
	// appropriate actions before that happens.
	NotReady() error
}

// ConfigWatcher is an interface that should be implemented by an extension that
// wishes to be notified of the Collector's effective configuration.
type ConfigWatcher interface {
	// NotifyConfig notifies the extension of the Collector's current effective configuration.
	// The extension owns the `confmap.Conf`. Callers must ensure that it's safe for
	// extensions to store the `conf` pointer and use it concurrently with any other
	// instances of `conf`.
	NotifyConfig(ctx context.Context, conf *confmap.Conf) error
}
