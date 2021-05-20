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

package storagetest

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStorageHost(t *testing.T) {
	host := NewStorageHost(t, newTempDir(t), "test")
	require.Equal(t, 1, len(host.GetExtensions()))

	hostWithTwo := NewStorageHost(t, newTempDir(t), "one", "two")
	require.Equal(t, 2, len(hostWithTwo.GetExtensions()))
}

func newTempDir(tb testing.TB) string {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(tb, err)
	tb.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}
