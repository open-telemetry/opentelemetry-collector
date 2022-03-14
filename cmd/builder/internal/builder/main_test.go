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

package builder

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDefault(t *testing.T) {
	require.NoError(t, Generate(NewDefaultConfig()))
}

func TestGenerateInvalidCollectorVersion(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Distribution.OtelColVersion = "invalid"
	err := Generate(cfg)
	require.NoError(t, err)
}

func TestGenerateInvalidOutputPath(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Distribution.OutputPath = "/invalid"
	err := Generate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create output path")
}

func TestGenerateAndCompileDefault(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "default")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cfg := NewDefaultConfig()
	cfg.Distribution.OutputPath = dir

	// we override this version, otherwise this would break during releases
	cfg.Distribution.OtelColVersion = "0.38.0"

	assert.NoError(t, cfg.Validate())
	require.NoError(t, GenerateAndCompile(cfg))
}
