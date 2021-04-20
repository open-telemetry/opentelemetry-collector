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

package batchprocessor

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// splitLogs removes logrecords from the input data and returns a new data of the specified size.
func splitLogs(size int, toSplit pdata.Logs) pdata.Logs {
	if toSplit.LogRecordCount() <= size {
		return toSplit
	}
	copiedLogs := 0
	result := pdata.NewLogs()
	rls := toSplit.ResourceLogs()
	result.ResourceLogs().Resize(rls.Len())
	rlsCount := 0
	for i := rls.Len() - 1; i >= 0; i-- {
		rlsCount++
		rl := rls.At(i)
		destRl := result.ResourceLogs().At(result.ResourceLogs().Len() - 1 - i)
		rl.Resource().CopyTo(destRl.Resource())

		for j := rl.InstrumentationLibraryLogs().Len() - 1; j >= 0; j-- {
			instLogs := rl.InstrumentationLibraryLogs().At(j)
			destInstLogs := destRl.InstrumentationLibraryLogs().AppendEmpty()
			instLogs.InstrumentationLibrary().CopyTo(destInstLogs.InstrumentationLibrary())

			if size-copiedLogs >= instLogs.Logs().Len() {
				destInstLogs.Logs().Resize(instLogs.Logs().Len())
			} else {
				destInstLogs.Logs().Resize(size - copiedLogs)
			}
			for k, destIdx := instLogs.Logs().Len()-1, 0; k >= 0 && copiedLogs < size; k, destIdx = k-1, destIdx+1 {
				log := instLogs.Logs().At(k)
				log.CopyTo(destInstLogs.Logs().At(destIdx))
				copiedLogs++
				// remove log
				instLogs.Logs().Resize(instLogs.Logs().Len() - 1)
			}
			if instLogs.Logs().Len() == 0 {
				rl.InstrumentationLibraryLogs().Resize(rl.InstrumentationLibraryLogs().Len() - 1)
			}
			if copiedLogs == size {
				result.ResourceLogs().Resize(rlsCount)
				return result
			}
		}
		if rl.InstrumentationLibraryLogs().Len() == 0 {
			rls.Resize(rls.Len() - 1)
		}
	}
	result.ResourceLogs().Resize(rlsCount)
	return result
}
