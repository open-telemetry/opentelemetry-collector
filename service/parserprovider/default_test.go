// Copyright The OpenTelemetry Authors
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

package parserprovider

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestDefaultMapProvider(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--config=testdata/default-config.yaml",
		"",
	})
	require.NoError(t, err)
	mp := NewDefaultMapProvider()
	require.NotNil(t, mp)
	var cm *config.Map
	cm, err = mp.Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cm)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
exporters:
  otlp:
    endpoint: "localhost:4317"`))
	require.NoError(t, err)
	assert.Equal(t, expectedMap, cm)

	assert.NoError(t, mp.Close(context.Background()))
}

func TestDefaultMapProvider_AddNewConfig(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--config=testdata/default-config.yaml",
		"--set=processors.batch.timeout=2s",
	})
	require.NoError(t, err)
	mp := NewDefaultMapProvider()
	require.NotNil(t, mp)
	var cm *config.Map
	cm, err = mp.Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cm)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
    timeout: 2s
exporters:
  otlp:
    endpoint: "localhost:4317"`))
	require.NoError(t, err)
	assert.Equal(t, expectedMap, cm)

	assert.NoError(t, mp.Close(context.Background()))
}

func TestDefaultMapProvider_OverwriteConfig(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--config=testdata/default-config.yaml",
		"--set=processors.batch.timeout=2s",
		"--set=exporters.otlp.endpoint=localhost:1234",
	})
	require.NoError(t, err)
	mp := NewDefaultMapProvider()
	require.NotNil(t, mp)
	var cm *config.Map
	cm, err = mp.Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cm)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
    timeout: 2s
exporters:
  otlp:
    endpoint: "localhost:1234"`))
	require.NoError(t, err)
	assert.Equal(t, expectedMap, cm)

	assert.NoError(t, mp.Close(context.Background()))
}

func TestDefaultMapProvider_InvalidFile(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--config=testdata/otelcol-config.yaml",
	})
	require.NoError(t, err)
	mp := NewDefaultMapProvider()
	require.NotNil(t, mp)
	_, err = mp.Get(context.Background())
	require.Error(t, err)

	assert.NoError(t, mp.Close(context.Background()))
}

func TestDefaultMapProvider_EmptyFileName(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{})
	require.NoError(t, err)
	mp := NewDefaultMapProvider()
	require.NotNil(t, mp)
	_, err = mp.Get(context.Background())
	require.Error(t, err)

	assert.NoError(t, mp.Close(context.Background()))
}
