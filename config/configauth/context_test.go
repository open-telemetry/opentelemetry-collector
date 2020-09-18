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

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubjectFromContext(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), subjectKey, "my-subject")

	// test
	sub, ok := SubjectFromContext(ctx)

	// verify
	assert.Equal(t, "my-subject", sub)
	assert.True(t, ok)
}

func TestSubjectFromContextNotPresent(t *testing.T) {
	// prepare
	ctx := context.Background()

	// test
	sub, ok := SubjectFromContext(ctx)

	// verify
	assert.False(t, ok)
	assert.Empty(t, sub)
}

func TestSubjectFromContextWrongType(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), subjectKey, 123)

	// test
	sub, ok := SubjectFromContext(ctx)

	// verify
	assert.False(t, ok)
	assert.Empty(t, sub)
}

func TestGroupsFromContext(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), groupsKey, []string{"my-groups"})

	// test
	groups, ok := GroupsFromContext(ctx)

	// verify
	assert.Equal(t, []string{"my-groups"}, groups)
	assert.True(t, ok)
}

func TestGroupsFromContextNotPresent(t *testing.T) {
	// prepare
	ctx := context.Background()

	// test
	sub, ok := GroupsFromContext(ctx)

	// verify
	assert.False(t, ok)
	assert.Empty(t, sub)
}

func TestGroupsFromContextWrongType(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), subjectKey, 123)

	// test
	sub, ok := GroupsFromContext(ctx)

	// verify
	assert.False(t, ok)
	assert.Empty(t, sub)
}
