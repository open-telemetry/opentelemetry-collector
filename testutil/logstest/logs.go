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

package logstest

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

type Log struct {
	Timestamp  int64
	Body       pdata.AttributeValue
	Attributes map[string]pdata.AttributeValue
}

// A convenience function for constructing logs for tests in a way that is
// relatively easy to read and write declaratively compared to the highly
// imperative and verbose method of using pdata directly.
// Attributes are sorted by key name.
func Logs(recs ...Log) pdata.Logs {
	out := pdata.NewLogs()

	logs := out.ResourceLogs()

	logs.Resize(1)
	rls := logs.At(0)

	rls.InstrumentationLibraryLogs().Resize(1)
	logSlice := rls.InstrumentationLibraryLogs().At(0).Logs()

	logSlice.Resize(len(recs))
	for i := range recs {
		l := logSlice.At(i)
		recs[i].Body.CopyTo(l.Body())
		l.SetTimestamp(pdata.Timestamp(recs[i].Timestamp))
		l.Attributes().InitFromMap(recs[i].Attributes)
		l.Attributes().Sort()
	}

	return out
}
