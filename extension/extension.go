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

// Package extension defines service extensions that can be added to the OpenTelemetry
// service but that not interact if the data pipelines, but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
package extension

// Host represents the entity where the extension is being hosted.
// It is used to allow communication between the extension and its host.
type Host interface {
	// ReportFatalError is used to report to the host that the extension
	// encountered a fatal error (i.e.: an error that the instance can't recover
	// from) after its start function had already returned.
	ReportFatalError(err error)
}

// ServiceExtension is the interface for objects hosted by the OpenTelemetry Service that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type ServiceExtension interface {
	// Start the ServiceExtension object hosted by the given host. At this point in the
	// process life-cycle the receivers are not started and the host did not
	// receive any data yet.
	Start(host Host) error

	// Shutdown the ServiceExtension instance. This happens after the pipelines were
	// shutdown.
	Shutdown() error
}

// PipelineWatcher is an extra interface for ServiceExtension hosted by the OpenTelemetry
// Service that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the ServiceExtension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (notice that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the ServiceExtension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the ServiceExtension can take any
	// appropriate action before that happens.
	NotReady() error
}
