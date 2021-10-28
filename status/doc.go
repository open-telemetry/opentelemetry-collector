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

// Package status contains structured status codes for use in the
// collector's own logs, to help with diagnosis. This is expected to
// be especially useful for automated tools that need to make sense of
// logs. The status definitions are the same as those used in gRPC, but
// copied in order to avoid the dependency.
package status // import "go.opentelemetry.io/collector/status"
