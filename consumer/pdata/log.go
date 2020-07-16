// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdata

import otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"

// NewResourceLogsSliceFromOrig creates ResourceLogsSlice from otlplogs.ResourceLogs.
// This function simply makes generated newResourceLogsSlice() function publicly
// available for internal.data.Log to call. We intentionally placed data.Log in the
// internal package so that it is not available publicly while it is experimental.
// Once the experiment is over data.Log should move to this package (pdata) and
// NewResourceLogsSliceFromOrig function will no longer be needed.
func NewResourceLogsSliceFromOrig(orig *[]*otlplogs.ResourceLogs) ResourceLogsSlice {
	return ResourceLogsSlice{orig}
}
