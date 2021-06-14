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
	"github.com/prometheus/prometheus/pkg/labels"
)

// stalenessStore tracks metrics/labels that appear between scrapes, the current and last scrape.
// The labels that appear only in the previous scrape are considered stale and for those, we
// issue a staleness marker aka a special NaN value.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/3413
type stalenessStore struct {
	currentHashes  map[uint64]bool
	previousHashes map[uint64]bool
	previous       []labels.Labels
	current        []labels.Labels
}

func newStalenessStore() *stalenessStore {
	return &stalenessStore{
		previousHashes: make(map[uint64]bool),
		currentHashes:  make(map[uint64]bool),
	}
}

// refresh copies over all the current values to previous, and prepares.
// refresh must be called before every new scrape.
func (ss *stalenessStore) refresh() {
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
	hash := lbl.Hash()
	return ss.previousHashes[hash] && !ss.currentHashes[hash]
}

// markAsCurrentlySeen adds lbl to the manifest of labels seen in the current scrape.
// This method should be called before refresh, but during a scrape whenever labels are encountered.
func (ss *stalenessStore) markAsCurrentlySeen(lbl labels.Labels) {
	ss.currentHashes[lbl.Hash()] = true
	ss.current = append(ss.current, lbl)
}

// emitStaleLabels returns the labels that were previously seen in
// the prior scrape, but are not currently present in this scrape cycle.
func (ss *stalenessStore) emitStaleLabels() (stale []labels.Labels) {
	for _, labels := range ss.previous {
		hash := labels.Hash()
		if ok := ss.currentHashes[hash]; !ok {
			stale = append(stale, labels)
		}
	}
	return stale
}
