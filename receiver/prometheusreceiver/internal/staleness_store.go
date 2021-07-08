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

import (
	"math"
	"sync"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/value"
)

// Prometheus uses a special NaN to record staleness as per
// https://github.com/prometheus/prometheus/blob/67dc912ac8b24f94a1fc478f352d25179c94ab9b/pkg/value/value.go#L24-L28
var stalenessSpecialValue = math.Float64frombits(value.StaleNaN)

// stalenessStore tracks metrics/labels that appear between scrapes, the current and last scrape.
// The labels that appear only in the previous scrape are considered stale and for those, we
// issue a staleness marker aka a special NaN value.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/3413
type stalenessStore struct {
	mu             sync.Mutex // mu protects all the fields below.
	currentHashes  map[uint64]int64
	previousHashes map[uint64]int64
	previous       []labels.Labels
	current        []labels.Labels
}

func newStalenessStore() *stalenessStore {
	return &stalenessStore{
		previousHashes: make(map[uint64]int64),
		currentHashes:  make(map[uint64]int64),
	}
}

// refresh copies over all the current values to previous, and prepares.
// refresh must be called before every new scrape.
func (ss *stalenessStore) refresh() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// 1. Clear ss.previousHashes firstly. Please don't edit
	// this map clearing idiom as it ensures speed.
	// See:
	// * https://github.com/golang/go/issues/20138
	// * https://github.com/golang/go/commit/aee71dd70b3779c66950ce6a952deca13d48e55e
	for hash := range ss.previousHashes {
		delete(ss.previousHashes, hash)
	}
	// 2. Copy over ss.currentHashes to ss.previousHashes.
	for hash := range ss.currentHashes {
		ss.previousHashes[hash] = ss.currentHashes[hash]
	}
	// 3. Clear ss.currentHashes, with the map clearing idiom for speed.
	// See:
	// * https://github.com/golang/go/issues/20138
	// * https://github.com/golang/go/commit/aee71dd70b3779c66950ce6a952deca13d48e55e
	for hash := range ss.currentHashes {
		delete(ss.currentHashes, hash)
	}
	// 4. Copy all the prior labels from what was previously ss.current.
	ss.previous = ss.current
	// 5. Clear ss.current to make for another cycle.
	ss.current = nil
}

// isStale returns whether lbl was seen only in the previous scrape and not the current.
func (ss *stalenessStore) isStale(lbl labels.Labels) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	hash := lbl.Hash()
	_, inPrev := ss.previousHashes[hash]
	_, inCurrent := ss.currentHashes[hash]
	return inPrev && !inCurrent
}

// markAsCurrentlySeen adds lbl to the manifest of labels seen in the current scrape.
// This method should be called before refresh, but during a scrape whenever labels are encountered.
func (ss *stalenessStore) markAsCurrentlySeen(lbl labels.Labels, seenAtMs int64) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.currentHashes[lbl.Hash()] = seenAtMs
	ss.current = append(ss.current, lbl)
}

type staleEntry struct {
	labels   labels.Labels
	seenAtMs int64
}

// emitStaleLabels returns the labels that were previously seen in
// the prior scrape, but are not currently present in this scrape cycle.
func (ss *stalenessStore) emitStaleLabels() (stale []*staleEntry) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	for _, labels := range ss.previous {
		hash := labels.Hash()
		if _, ok := ss.currentHashes[hash]; !ok {
			stale = append(stale, &staleEntry{seenAtMs: ss.previousHashes[hash], labels: labels})
		}
	}
	return stale
}
