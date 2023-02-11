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

package configtls

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestLoader struct{}

func (r TestLoader) loadClientCAFile() (*x509.CertPool, error) {
	return nil, nil
}

func TestCannotShutdownIfNotWatching(t *testing.T) {
	tmpClientCAsFilePath := createTempFile(t)

	loader := TestLoader{}
	reloader, _ := newClientCAsReloader(tmpClientCAsFilePath, loader)

	err := reloader.shutdown()
	assert.Error(t, err)
}

func TestCannotStartIfAlreadyWatching(t *testing.T) {
	tmpClientCAsFilePath := createTempFile(t)

	loader := TestLoader{}
	reloader, _ := newClientCAsReloader(tmpClientCAsFilePath, loader)

	err := reloader.startWatching()
	assert.NoError(t, err)

	err = reloader.startWatching()
	assert.Error(t, err)

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func createTempFile(t *testing.T) string {
	tmpCa, err := os.CreateTemp("", "clientCAs.crt")
	assert.NoError(t, err)
	tmpCaPath, err := filepath.Abs(tmpCa.Name())
	assert.NoError(t, err)
	assert.NoError(t, tmpCa.Close())
	return tmpCaPath
}
