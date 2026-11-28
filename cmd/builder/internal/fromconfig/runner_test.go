// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeListed(t *testing.T) {
	got, err := decodeListed(strings.NewReader(`{"Path":"a","Versions":["v0.1.0","v0.2.0"]}
{"Path":"b","Error":{"Err":"404 Not Found"}}
{"Path":"c"}`))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.True(t, got[0].exists())
	assert.False(t, got[1].exists())
	assert.Equal(t, "404 Not Found", got[1].Error.Err)
	assert.False(t, got[2].exists())

	_, err = decodeListed(strings.NewReader(`{"Path":`))
	require.Error(t, err)
}

func TestGoRunnerModule(t *testing.T) {
	g, err := newGoRunner()
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(g.dir, "go.mod"))

	require.NoError(t, g.close())
	assert.NoDirExists(t, g.dir)
}
