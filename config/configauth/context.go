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

package configauth

import "context"

var (
	subjectKey = subjectType{}
	groupsKey  = groupsType{}
)

type subjectType struct{}
type groupsType struct{}

// SubjectFromContext returns a list of groups the subject in the context belongs to
func SubjectFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(subjectKey).(string)
	return value, ok
}

// GroupsFromContext returns a list of groups the subject in the context belongs to
func GroupsFromContext(ctx context.Context) ([]string, bool) {
	value, ok := ctx.Value(groupsKey).([]string)
	return value, ok
}
