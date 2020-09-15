// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterlabel

// Config configures a filterlabel Matcher. It is nested and can be up to three
// levels deep. The first level matches against metric names, the second level
// against label keys, and the third level against label values. The second and
// third levels are optional.
type Config struct {
	str       string
	matchType MatchType
	next      []Config
}

// MatchType indicates the type of string matching to perform: strict or regexp.
type MatchType string

const (
	// The full string must match exactly
	MatchTypeStrict MatchType = "strict"
	// A partial-string regexp must match
	MatchTypeRegexp MatchType = "regexp"
)
