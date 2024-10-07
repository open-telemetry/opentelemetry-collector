// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtimeout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalTextValid(t *testing.T) {
	for _, valid := range []string{
		string(PolicySustain),
		string(PolicyAbort),
		string(PolicyIgnore),
		string(policyUnset),
		string(PolicyDefault),
	} {
		t.Run(valid, func(t *testing.T) {
			pol := Policy(valid)
			require.NoError(t, pol.Validate())
			assert.Equal(t, Policy(valid), pol)
		})
	}
}

func TestUnmarshalTextInvalid(t *testing.T) {
	for _, invalid := range []string{
		"unknown",
		"invalid",
		"etc",
	} {
		t.Run(invalid, func(t *testing.T) {
			pol := Policy(invalid)
			require.Error(t, pol.Validate())
			assert.Equal(t, Policy(invalid), pol)
		})
	}
}

func TestTimeoutDefaultPolicy(t *testing.T) {
	// The default behavior matches the legacy exporterhelper code base.
	require.Equal(t, PolicySustain, PolicyDefault)
}
