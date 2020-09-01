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

package internal

import otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"

// OtlpLogsWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Logs data type to be callable by
// any code outside of this module.
type OtlpLogsWrapper struct {
	Orig *[]*otlplogs.ResourceLogs
}

func LogsToOtlp(l OtlpLogsWrapper) []*otlplogs.ResourceLogs {
	return *l.Orig
}

func LogsFromOtlp(logs []*otlplogs.ResourceLogs) OtlpLogsWrapper {
	return OtlpLogsWrapper{Orig: &logs}
}
