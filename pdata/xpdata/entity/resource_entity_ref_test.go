// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestResourceEntityRefs(t *testing.T) {
	res := internal.GenerateTestResource()
	assert.EqualValues(t, internal.GenerateTestEntityRefSlice(), ResourceEntityRefs(pcommon.Resource(res)))
}
