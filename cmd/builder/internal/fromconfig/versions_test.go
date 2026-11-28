// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const otelcolGoMod = `module go.opentelemetry.io/collector/otelcol

go 1.25

require (
	go.opentelemetry.io/collector/component v1.68.0
	go.opentelemetry.io/collector/component/componentstatus v0.162.0
	go.opentelemetry.io/collector/service v0.162.0
)

replace go.opentelemetry.io/collector/component => ../component
`

func TestParseVersions(t *testing.T) {
	got, err := parseVersions("go.mod", []byte(otelcolGoMod))
	require.NoError(t, err)
	assert.Equal(t, Versions{Beta: "v0.162.0", Stable: "v1.68.0"}, got)
}

func TestParseVersionsErrors(t *testing.T) {
	_, err := parseVersions("go.mod", []byte("module x\n\nrequire go.opentelemetry.io/collector/component v1.68.0\n"))
	require.ErrorContains(t, err, "does not require both")

	_, err = parseVersions("go.mod", []byte("not a go.mod"))
	require.ErrorContains(t, err, "failed to parse")
}

func TestSelectVersion(t *testing.T) {
	v := Versions{Beta: "v0.162.0", Stable: "v1.68.0"}

	got, ok := v.selectVersion([]string{"v0.161.0", "v0.162.0"})
	assert.True(t, ok)
	assert.Equal(t, "v0.162.0", got)

	got, ok = v.selectVersion([]string{"v1.67.0", "v1.68.0"})
	assert.True(t, ok)
	assert.Equal(t, "v1.68.0", got)

	// Versions of another release are never selected.
	_, ok = v.selectVersion([]string{"v0.161.0", "v1.67.0", "v0.163.0"})
	assert.False(t, ok)
	_, ok = v.selectVersion(nil)
	assert.False(t, ok)
}
