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

package configmapprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYamlProvider_Empty(t *testing.T) {
	sp := NewYAML()
	_, err := sp.Retrieve(context.Background(), "", nil)
	assert.Error(t, err)
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider_InvalidValue(t *testing.T) {
	sp := NewYAML()
	_, err := sp.Retrieve(context.Background(), "yaml::2s", nil)
	assert.Error(t, err)
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider(t *testing.T) {
	sp := NewYAML()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch::timeout: 2s", nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"processors": map[string]interface{}{
			"batch": map[string]interface{}{
				"timeout": "2s",
			},
		},
	}, ret.Map.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider_NamedComponent(t *testing.T) {
	sp := NewYAML()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch/foo::timeout: 3s", nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"processors": map[string]interface{}{
			"batch/foo": map[string]interface{}{
				"timeout": "3s",
			},
		},
	}, ret.Map.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider_MapEntry(t *testing.T) {
	sp := NewYAML()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors: {batch/foo::timeout: 3s, batch::timeout: 2s}", nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"processors": map[string]interface{}{
			"batch/foo": map[string]interface{}{
				"timeout": "3s",
			},
			"batch": map[string]interface{}{
				"timeout": "2s",
			},
		},
	}, ret.Map.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider_NewLine(t *testing.T) {
	sp := NewYAML()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch/foo::timeout: 3s\nprocessors::batch::timeout: 2s", nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"processors": map[string]interface{}{
			"batch/foo": map[string]interface{}{
				"timeout": "3s",
			},
			"batch": map[string]interface{}{
				"timeout": "2s",
			},
		},
	}, ret.Map.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestYamlProvider_DotSeparator(t *testing.T) {
	sp := NewYAML()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors.batch.timeout: 4s", nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"processors.batch.timeout": "4s"}, ret.Map.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}
