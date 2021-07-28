// Copyright Splunk, Inc.
// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configprovider

import (
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/testutil"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func TestConfigServer_RequireEnvVar(t *testing.T) {
	initial := map[string]interface{}{
		"minimal": "config",
	}

	cs := newConfigServer(zap.NewNop(), initial, initial)
	require.NotNil(t, cs)

	require.NoError(t, cs.start())
	t.Cleanup(func() {
		require.NoError(t, cs.shutdown())
	})

	client := &http.Client{}
	path := "/debug/configz/initial"
	_, err := client.Get("http://" + defaultConfigServerEndpoint + path)
	assert.Error(t, err)
}

func TestConfigServer_EnvVar(t *testing.T) {
	alternativePort := strconv.FormatUint(uint64(testutil.GetAvailablePort(t)), 10)
	require.NoError(t, os.Setenv(configServerEnabledEnvVar, "true"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv(configServerEnabledEnvVar))
	})

	tests := []struct {
		name          string
		portEnvVar    string
		endpoint      string
		setPortEnvVar bool
		serverDown    bool
	}{
		{
			name: "default",
		},
		{
			name:          "disable_server",
			setPortEnvVar: true, // Explicitly setting it to empty to disable the server.
			serverDown:    true,
		},
		{
			name:       "change_port",
			portEnvVar: alternativePort,
			endpoint:   "http://localhost:" + alternativePort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := map[string]interface{}{
				"key": "value",
			}

			if tt.portEnvVar != "" || tt.setPortEnvVar {
				require.NoError(t, os.Setenv(configServerPortEnvVar, tt.portEnvVar))
				defer func() {
					assert.NoError(t, os.Unsetenv(configServerPortEnvVar))
				}()
			}

			cs := newConfigServer(zap.NewNop(), initial, initial)
			require.NoError(t, cs.start())
			defer func() {
				assert.NoError(t, cs.shutdown())
			}()

			endpoint := tt.endpoint
			if endpoint == "" {
				endpoint = "http://" + defaultConfigServerEndpoint
			}

			path := "/debug/configz/initial"
			if tt.serverDown {
				client := &http.Client{}
				_, err := client.Get(endpoint + path)
				assert.Error(t, err)
				return
			}

			client := &http.Client{}
			resp, err := client.Get(endpoint + path)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, "unsuccessful zpage %q GET", path)
		})
	}
}

func TestConfigServer_Serve(t *testing.T) {
	require.NoError(t, os.Setenv(configServerEnabledEnvVar, "true"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv(configServerEnabledEnvVar))
	})

	initial := map[string]interface{}{
		"field":   "not_redacted",
		"api_key": "not_redacted_on_initial",
		"int":     42,
		"map": map[interface{}]interface{}{
			"k0":       true,
			"k1":       -1,
			"password": "$ENV_VAR",
		},
	}
	effective := map[string]interface{}{
		"field":   "not_redacted",
		"api_key": "<redacted>",
		"int":     42,
		"map": map[interface{}]interface{}{
			"k0":       true,
			"k1":       -1,
			"password": "<redacted>",
		},
	}

	cs := newConfigServer(zap.NewNop(), initial, initial)
	require.NotNil(t, cs)

	require.NoError(t, cs.start())
	t.Cleanup(func() {
		require.NoError(t, cs.shutdown())
	})

	// Test for the pages to be actually valid YAML files.
	assertValidYAMLPages(t, initial, "/debug/configz/initial")
	assertValidYAMLPages(t, effective, "/debug/configz/effective")
}

func assertValidYAMLPages(t *testing.T, expected map[string]interface{}, path string) {
	url := "http://" + defaultConfigServerEndpoint + path

	client := &http.Client{}

	// Anything other the GET should return 405.
	resp, err := client.Head(url)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())

	resp, err = client.Get(url)
	if !assert.NoError(t, err, "error retrieving zpage at %q", path) {
		return
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "unsuccessful zpage %q GET", path)
	t.Cleanup(func() {
		assert.NoError(t, resp.Body.Close())
	})

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var unmarshalled map[string]interface{}
	require.NoError(t, yaml.Unmarshal(respBytes, &unmarshalled))

	assert.Equal(t, expected, unmarshalled)
}
