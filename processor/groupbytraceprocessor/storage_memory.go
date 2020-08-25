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
	"errors"
	"sync"

	"go.opentelemetry.io/collector/consumer/pdata"
)

var errStorageNilResourceSpans = errors.New("the provided trace is invalid (nil)")

type memoryStorage struct {
	sync.RWMutex
	content map[string][]pdata.ResourceSpans
}

var _ storage = (*memoryStorage)(nil)

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{
		content: make(map[string][]pdata.ResourceSpans),
	}
}

func (st *memoryStorage) createOrAppend(traceID pdata.TraceID, rs pdata.ResourceSpans) error {
	if rs.IsNil() {
		return errStorageNilResourceSpans
	}

	sTraceID := traceID.String()

	st.Lock()
	if _, ok := st.content[sTraceID]; !ok {
		st.content[sTraceID] = []pdata.ResourceSpans{}
	}

	newRS := pdata.NewResourceSpans()
	rs.CopyTo(newRS)
	st.content[sTraceID] = append(st.content[sTraceID], newRS)

	st.Unlock()

	return nil
}
func (st *memoryStorage) get(traceID pdata.TraceID) ([]pdata.ResourceSpans, error) {
	sTraceID := traceID.String()

	st.RLock()
	rss, ok := st.content[sTraceID]
	if !ok {
		return nil, nil
	}

	result := []pdata.ResourceSpans{}
	for _, rs := range rss {
		newRS := pdata.NewResourceSpans()
		rs.CopyTo(newRS)
		result = append(result, newRS)
	}
	st.RUnlock()

	return result, nil
}

// delete will return a reference to a ResourceSpans. Changes to the returned object may not be applied
// to the version in the storage.
func (st *memoryStorage) delete(traceID pdata.TraceID) ([]pdata.ResourceSpans, error) {
	sTraceID := traceID.String()

	st.Lock()
	rss := st.content[sTraceID]
	result := []pdata.ResourceSpans{}
	for _, rs := range rss {
		newRS := pdata.NewResourceSpans()
		rs.CopyTo(newRS)
		result = append(result, newRS)
	}
	delete(st.content, sTraceID)
	st.Unlock()

	return result, nil
}

func (st *memoryStorage) count() int {
	st.RLock()
	defer st.RUnlock()
	return len(st.content)
}
