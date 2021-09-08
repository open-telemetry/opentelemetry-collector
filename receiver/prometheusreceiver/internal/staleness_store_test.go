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
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestStalenessStore(t *testing.T) {
	ss := newStalenessStore()
	require.NotNil(t, ss.previousHashes)
	require.Zero(t, len(ss.previousHashes))
	require.NotNil(t, ss.currentHashes)
	require.Zero(t, len(ss.currentHashes))

	lbl1 := labels.Labels{
		{Name: "__name__", Value: "lbl1"},
		{Name: "a", Value: "1"},
	}
	lbl2 := labels.Labels{
		{Name: "__name__", Value: "lbl2"},
		{Name: "b", Value: "1"},
	}
	ss.markAsCurrentlySeen(lbl1)
	require.Nil(t, ss.emitStaleLabels())
	require.False(t, ss.isStale(lbl1))
	require.False(t, ss.isStale(lbl2))

	// Now refresh, the case of a new scrape.
	// Without having marked lbl1 as being current, it should be reported as stale.
	ss.refresh()
	require.True(t, ss.isStale(lbl1))
	require.False(t, ss.isStale(lbl2))
	// .previous should have been the prior contents of current and current should be nil.
	require.Equal(t, ss.previous, []labels.Labels{lbl1})
	require.Nil(t, ss.current)

	// After the next refresh cycle, we shouldn't have any stale labels.
	ss.refresh()
	require.False(t, ss.isStale(lbl1))
	require.False(t, ss.isStale(lbl2))
}
