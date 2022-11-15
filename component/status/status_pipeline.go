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

package status

// PipelineReadiness represents an enumeration of pipeline statuses
type PipelineReadiness int

const (
	// PipelineReady indicates the pipeline is ready
	PipelineReady PipelineReadiness = iota
	// PipelineNotReady indicates the pipeline is not ready
	PipelineNotReady
)

// PipelineStatusFunc is a function to be called when the collector pipeline changes states
type PipelineStatusFunc func(st PipelineReadiness) error

var noopPipelineStatusFunc = func(st PipelineReadiness) error { return nil }
