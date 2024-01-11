// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque // import "go.opentelemetry.io/collector/config/configopaque"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringMarshalText(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	for _, example := range examples {
		opaque, err := example.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, "[REDACTED]", string(opaque))
	}
}

func TestStringFmt(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	for _, example := range examples {
		// nolint S1025
		assert.Panics(t, func() { fmt.Sprintf("%s", example) })
		// converting to a string allows to output as an unredacted string still:
		// nolint S1025
		assert.Equal(t, string(example), fmt.Sprintf("%s", string(example)))
	}
}
