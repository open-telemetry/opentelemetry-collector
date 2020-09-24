// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupbytraceprocessor

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// storage is an abstraction for the span storage used by the groupbytrace processor.
// Implementations should be safe for concurrent use.
type storage interface {
	// createOrAppend will check whether the given trace ID is already in the storage and
	// will either append the given spans to the existing record, or create a new trace with
	// the given resource spans
	createOrAppend(pdata.TraceID, pdata.ResourceSpans) error

	// get will retrieve the trace based on the given trace ID, returning nil in case a trace
	// cannot be found
	get(pdata.TraceID) ([]pdata.ResourceSpans, error)

	// delete will remove the trace based on the given trace ID, returning the trace that was removed,
	// or nil in case a trace cannot be found
	delete(pdata.TraceID) ([]pdata.ResourceSpans, error)

	// start gives the storage the opportunity to initialize any resources or procedures
	start() error

	// shutdown signals the storage that the processor is shutting down
	shutdown() error
}
