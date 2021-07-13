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

package auth

import "context"

var (
	rawKey     = rawType{}
	subjectKey = subjectType{}
	groupsKey  = groupsType{}
)

type rawType struct{}
type subjectType struct{}
type groupsType struct{}

// NewContextFromRaw creates a new context derived from the given context,
// adding the raw authentication string to the result.
func NewContextFromRaw(ctx context.Context, raw string) context.Context {
	return context.WithValue(ctx, rawKey, raw)
}

// NewContextFromSubject creates a new context derived from the given context,
// adding the subject to the result.
func NewContextFromSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, subjectKey, subject)
}

// NewContextFromMemberships creates a new context derived from the given context,
// adding the memberships to the result.
func NewContextFromMemberships(ctx context.Context, groups []string) context.Context {
	return context.WithValue(ctx, groupsKey, groups)
}

// RawFromContext returns the raw authentication string used to perform the authentication.
// It's typically the string value for a single HTTP header, such as the "Authentication" header.
// Example: "Basic ZGV2ZWxvcGVyOmN1cmlvdXM=".
func RawFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(rawKey).(string)
	return value, ok
}

// SubjectFromContext returns the subject that was extracted from the raw authentication string.
func SubjectFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(subjectKey).(string)
	return value, ok
}

// GroupsFromContext returns the list of groups the subject belongs.
func GroupsFromContext(ctx context.Context) ([]string, bool) {
	value, ok := ctx.Value(groupsKey).([]string)
	return value, ok
}
