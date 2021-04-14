// Copyright 2020 Splunk, Inc.
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

package vaultconfigsource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/experimental/configsource"
)

const (
	address        = "http://localhost:8200"
	token          = "dev_token"
	vaultContainer = "vault_tests"
	mongoContainer = "mongodb"

	// For the commands below whitespace is used to break the parameters into a correct slice of arguments.

	startVault = "docker run --rm -d -p 8200:8200 -e VAULT_DEV_ROOT_TOKEN_ID=" + token + " -e VAULT_TOKEN=" + token + " -e VAULT_ADDR=http://localhost:8200 --name=" + vaultContainer + " vault"
	stopVault  = "docker stop " + vaultContainer

	setupKVStore  = "docker exec " + vaultContainer + " vault kv put secret/kv k0=v0 k1=v1"
	updateKVStore = "docker exec " + vaultContainer + " vault kv put secret/kv k0=v0 k1=v1.1"

	startMongo            = "docker run --rm -d -p 27017:27017 --name=" + mongoContainer + " mongo"
	stopMongo             = "docker stop " + mongoContainer
	setupDatabaseStore    = "docker exec " + vaultContainer + " vault secrets enable database"
	setupMongoVaultPlugin = "docker exec " + vaultContainer + " vault write database/config/my-mongodb-database plugin_name=mongodb-database-plugin allowed_roles=my-role connection_url=mongodb://host.docker.internal:27017/admin username=\"admin\" password=\"\""
	setupMongoSecret      = "docker exec " + vaultContainer + " vault write database/roles/my-role db_name=my-mongodb-database creation_statements={\"db\":\"admin\",\"roles\":[{\"role\":\"readWrite\"},{\"role\":\"read\",\"db\":\"foo\"}]} default_ttl=2s max_ttl=6s"

	createKVVer1Store = "docker exec " + vaultContainer + " vault secrets enable -version=1 kv"
	setupKVVer1Store  = "docker exec " + vaultContainer + " vault kv put kv/my-secret ttl=8s my-value=s3cr3t"
	setupKVVer1NoTTL  = "docker exec " + vaultContainer + " vault kv put kv/my-secret ttl=0s my-value=s3cr3t"
)

func TestVaultSessionForKV(t *testing.T) {
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, setupKVStore)

	logger := zap.NewNop()
	config := Config{
		Endpoint:     address,
		Token:        token,
		Path:         "secret/data/kv",
		PollInterval: 2 * time.Second,
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	retrieved, err := s.Retrieve(context.Background(), "data.k0", nil)
	require.NoError(t, err)
	require.Equal(t, "v0", retrieved.Value().(string))

	retrievedMetadata, err := s.Retrieve(context.Background(), "metadata.version", nil)
	require.NoError(t, err)
	require.NotNil(t, retrievedMetadata.Value())

	require.NoError(t, s.RetrieveEnd(context.Background()))

	var watcherErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrieved.WatchForUpdate()
	}()

	require.NoError(t, s.Close(context.Background()))

	<-doneCh
	require.Equal(t, configsource.ErrSessionClosed, watcherErr)
}

func TestVaultPollingKVUpdate(t *testing.T) {
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, setupKVStore)

	logger := zap.NewNop()
	config := Config{
		Endpoint:     address,
		Token:        token,
		Path:         "secret/data/kv",
		PollInterval: 2 * time.Second,
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve key "k0"
	retrievedK0, err := s.Retrieve(context.Background(), "data.k0", nil)
	require.NoError(t, err)
	require.Equal(t, "v0", retrievedK0.Value().(string))

	// Retrieve key "k1"
	retrievedK1, err := s.Retrieve(context.Background(), "data.k1", nil)
	require.NoError(t, err)
	require.Equal(t, "v1", retrievedK1.Value().(string))

	// RetrieveEnd
	require.NoError(t, s.RetrieveEnd(context.Background()))

	// Only the first retrieved key provides a working watcher.
	require.Equal(t, configsource.ErrWatcherNotSupported, retrievedK1.WatchForUpdate())

	var watcherErr error
	var doneCh chan struct{}
	doneCh = make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedK0.WatchForUpdate()
	}()

	requireCmdRun(t, updateKVStore)

	// Wait for update.
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrValueUpdated)

	// Close current session.
	require.NoError(t, s.Close(context.Background()))

	// Create a new session and repeat the process.
	s, err = cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve key
	retrievedUpdatedK1, err := s.Retrieve(context.Background(), "data.k1", nil)
	require.NoError(t, err)
	require.Equal(t, "v1.1", retrievedUpdatedK1.Value().(string))

	// Wait for close.
	doneCh = make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedUpdatedK1.WatchForUpdate()
	}()

	require.NoError(t, s.Close(context.Background()))
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrSessionClosed)
}

func TestVaultRenewableSecret(t *testing.T) {
	// This test is based on the commands described at https://www.vaultproject.io/docs/secrets/databases/mongodb
	requireCmdRun(t, startMongo)
	defer requireCmdRun(t, stopMongo)
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, setupDatabaseStore)
	requireCmdRun(t, setupMongoVaultPlugin)
	requireCmdRun(t, setupMongoSecret)

	logger := zap.NewNop()
	config := Config{
		Endpoint:     address,
		Token:        token,
		Path:         "database/creds/my-role",
		PollInterval: 2 * time.Second,
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve key username, it is generated by vault no expected value.
	retrievedUser, err := s.Retrieve(context.Background(), "username", nil)
	require.NoError(t, err)

	// Retrieve key password, it is generated by vault no expected value.
	retrievedPwd, err := s.Retrieve(context.Background(), "password", nil)
	require.NoError(t, err)

	// RetrieveEnd
	require.NoError(t, s.RetrieveEnd(context.Background()))

	// Only the first retrieved key provides a working watcher.
	require.Equal(t, configsource.ErrWatcherNotSupported, retrievedPwd.WatchForUpdate())

	watcherErr := retrievedUser.WatchForUpdate()
	require.ErrorIs(t, watcherErr, configsource.ErrValueUpdated)

	// Close current session.
	require.NoError(t, s.Close(context.Background()))

	// Create a new session and repeat the process.
	s, err = cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve key username, it is generated by vault no expected value.
	retrievedUpdatedUser, err := s.Retrieve(context.Background(), "username", nil)
	require.NoError(t, err)
	require.NotEqual(t, retrievedUser.Value(), retrievedUpdatedUser.Value())

	// Retrieve password and check that it changed.
	retrievedUpdatedPwd, err := s.Retrieve(context.Background(), "password", nil)
	require.NoError(t, err)
	require.NotEqual(t, retrievedPwd.Value(), retrievedUpdatedPwd.Value())

	// Wait for close.
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedUpdatedUser.WatchForUpdate()
	}()

	runtime.Gosched()
	require.NoError(t, s.Close(context.Background()))
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrSessionClosed)
}

func TestVaultV1SecretWithTTL(t *testing.T) {
	// This test is based on the commands described at https://www.vaultproject.io/docs/secrets/kv/kv-v1
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, createKVVer1Store)
	requireCmdRun(t, setupKVVer1Store)

	logger := zap.NewNop()
	config := Config{
		Endpoint:     address,
		Token:        token,
		Path:         "kv/my-secret",
		PollInterval: 2 * time.Second,
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve value
	retrievedValue, err := s.Retrieve(context.Background(), "my-value", nil)
	require.NoError(t, err)
	require.Equal(t, "s3cr3t", retrievedValue.Value().(string))

	// RetrieveEnd
	require.NoError(t, s.RetrieveEnd(context.Background()))

	var watcherErr error
	var doneCh chan struct{}
	doneCh = make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedValue.WatchForUpdate()
	}()

	// Wait for update.
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrValueUpdated)

	// Close current session.
	require.NoError(t, s.Close(context.Background()))

	// Create a new session and repeat the process.
	s, err = cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve value
	retrievedValue, err = s.Retrieve(context.Background(), "my-value", nil)
	require.NoError(t, err)
	require.Equal(t, "s3cr3t", retrievedValue.Value().(string))

	// Wait for close.
	doneCh = make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedValue.WatchForUpdate()
	}()

	require.NoError(t, s.Close(context.Background()))
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrSessionClosed)
}

func TestVaultV1NonWatchableSecret(t *testing.T) {
	// This test is based on the commands described at https://www.vaultproject.io/docs/secrets/kv/kv-v1
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, createKVVer1Store)
	requireCmdRun(t, setupKVVer1NoTTL)

	logger := zap.NewNop()
	config := Config{
		Endpoint:     address,
		Token:        token,
		Path:         "kv/my-secret",
		PollInterval: 2 * time.Second,
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	// Retrieve value
	retrievedValue, err := s.Retrieve(context.Background(), "my-value", nil)
	require.NoError(t, err)
	require.Equal(t, "s3cr3t", retrievedValue.Value().(string))

	// RetrieveEnd
	require.NoError(t, s.RetrieveEnd(context.Background()))

	var watcherErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		watcherErr = retrievedValue.WatchForUpdate()
	}()

	// Wait for update.
	<-doneCh
	require.ErrorIs(t, watcherErr, configsource.ErrWatcherNotSupported)

	// Close current session.
	require.NoError(t, s.Close(context.Background()))
}

func TestVaultRetrieveErrors(t *testing.T) {
	requireCmdRun(t, startVault)
	defer requireCmdRun(t, stopVault)
	requireCmdRun(t, setupKVStore)

	ctx := context.Background()

	tests := []struct {
		err      error
		name     string
		path     string
		token    string
		selector string
	}{
		{
			name:  "bad_token",
			path:  "secret/data/kv",
			token: "bad_test_token",
			err:   &errClientRead{},
		},
		{
			name: "non_existent_path",
			path: "made_up_path/data/kv",
			err:  &errNilSecret{},
		},
		{
			name: "v2_missing_data_on_path",
			path: "secret/kv",
			err:  &errNilSecretData{},
		},
		{
			name:     "bad_selector",
			path:     "secret/data/kv",
			selector: "data.missing",
			err:      &errBadSelector{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testToken := token
			if tt.token != "" {
				testToken = tt.token
			}

			logger := zap.NewNop()
			config := Config{
				Endpoint:     address,
				Token:        testToken,
				Path:         tt.path,
				PollInterval: 2 * time.Second,
			}

			cfgSrc, err := newConfigSource(logger, &config)
			require.NoError(t, err)
			require.NotNil(t, cfgSrc)

			s, err := cfgSrc.NewSession(ctx)
			require.NoError(t, err)
			require.NotNil(t, s)

			defer func() {
				assert.NoError(t, s.Close(ctx))
			}()
			defer func() {
				assert.NoError(t, s.RetrieveEnd(ctx))
			}()

			r, err := s.Retrieve(ctx, tt.selector, nil)
			require.Error(t, err)
			require.IsType(t, tt.err, err)
			require.Nil(t, r)
		})
	}
}

func Test_vaultSession_extractVersionMetadata(t *testing.T) {
	tests := []struct {
		metadataMap map[string]interface{}
		expectedMd  *versionMetadata
		name        string
	}{
		{
			name: "typical",
			metadataMap: map[string]interface{}{
				"tsKey":  "2021-04-02T22:30:51.4733477Z",
				"verKey": json.Number("1"),
			},
			expectedMd: &versionMetadata{
				Timestamp: "2021-04-02T22:30:51.4733477Z",
				Version:   1,
			},
		},
		{
			name: "missing_expected_timestamp",
			metadataMap: map[string]interface{}{
				"otherKey": "2021-04-02T22:30:51.4733477Z",
				"verKey":   json.Number("1"),
			},
		},
		{
			name: "missing_expected_version",
			metadataMap: map[string]interface{}{
				"tsKey":    "2021-04-02T22:30:51.4733477Z",
				"otherKey": json.Number("1"),
			},
		},
		{
			name: "incorrect_version_format",
			metadataMap: map[string]interface{}{
				"tsKey":  "2021-04-02T22:30:51.4733477Z",
				"verKey": json.Number("not_a_number"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &vaultSession{
				logger: zap.NewNop(),
			}

			metadata := v.extractVersionMetadata(tt.metadataMap, "tsKey", "verKey")
			assert.Equal(t, tt.expectedMd, metadata)
		})
	}
}

func requireCmdRun(t *testing.T, cli string) {
	skipCheck(t)
	parts := strings.Split(cli, " ")
	cmd := exec.Command(parts[0], parts[1:]...) // #nosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	time.Sleep(500 * time.Millisecond)
	if err != nil {
		err = fmt.Errorf("cmd.Run() %s %v failed %w. stdout: %q stderr: %q", cmd.Path, cmd.Args, err, stdout.String(), stderr.String())
	}
	require.NoError(t, err)
}

func skipCheck(t *testing.T) {
	if s, ok := os.LookupEnv("RUN_VAULT_DOCKER_TESTS"); ok && s != "" {
		return
	}
	t.Skipf("Test must be explicitly enabled via 'RUN_VAULT_DOCKER_TESTS' environment variable.")
}
