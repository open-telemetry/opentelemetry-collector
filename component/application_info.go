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

package component

// ApplicationStartInfo is the information that is logged at the application start and
// passed into each component. This information can be overridden in custom builds.
type ApplicationStartInfo struct {
	// Executable file name, e.g. "otelcol".
	ExeName string

	// Long name, used e.g. in the logs.
	LongName string

	// Version string.
	Version string

	// Git hash of the source code.
	GitHash string
}

// DefaultApplicationStartInfo returns the default ApplicationStartInfo.
func DefaultApplicationStartInfo() ApplicationStartInfo {
	return ApplicationStartInfo{
		ExeName:  "otelcol",
		LongName: "OpenTelemetry Collector",
		Version:  "latest",
		GitHash:  "<NOT PROPERLY GENERATED>",
	}
}
