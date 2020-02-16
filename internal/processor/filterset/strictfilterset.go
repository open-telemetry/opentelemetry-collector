// Copyright 2020 OpenTelemetry Authors
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

package filterset

// strictFilterSet encapsulates a set of exact string match filters.
type strictFilterSet struct {
	filters map[string]bool
}

// sfsOption are options that mutate the strictFilterSet.
// rsOption is intentionally unexported to restrict the mutations possible.
type sfsOption func(*strictFilterSet)

// NewStrictFilterSet constructs a FilterSet of exact string matches.
func NewStrictFilterSet(filters []string, opts ...sfsOption) (FilterSet, error) {
	fs := &strictFilterSet{
		filters: map[string]bool{},
	}

	for _, o := range opts {
		o(fs)
	}

	if err := fs.addFilters(filters); err != nil {
		return nil, err
	}

	return fs, nil
}

// Matches returns true if the given string matches any of the FitlerSet's filters.
func (sfs *strictFilterSet) Matches(toMatch string) bool {
	_, ok := sfs.filters[toMatch]
	return ok
}

// addFilters all the given filters.
func (sfs *strictFilterSet) addFilters(filters []string) error {
	for _, f := range filters {
		sfs.filters[f] = true
	}

	return nil
}
