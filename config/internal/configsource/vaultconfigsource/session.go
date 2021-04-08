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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/internal/configsource"
)

var errInvalidPollInterval = errors.New("poll interval must be greater than zero")

// Error wrapper types to help with testability
type (
	errClientRead    struct{ error }
	errNilSecret     struct{ error }
	errNilSecretData struct{ error }
	errBadSelector   struct{ error }
)

// vaultSession implements the configsource.Session interface.
type vaultSession struct {
	logger *zap.Logger
	client *api.Client
	secret *api.Secret

	doneCh chan struct{}

	path string

	pollInterval time.Duration
}

var _ configsource.Session = (*vaultSession)(nil)

func (v *vaultSession) Retrieve(_ context.Context, selector string, _ interface{}) (configsource.Retrieved, error) {
	// By default assume that watcher is not supported. The exception will be the first
	// value read from the vault secret.
	watchForUpdateFn := watcherNotSupported

	if v.secret == nil {
		if err := v.readSecret(); err != nil {
			return nil, err
		}

		// The keys come all from the same secret so creating a watcher only for the
		// first it is fine.
		var err error
		watchForUpdateFn, err = v.buildWatcherFn()
		if err != nil {
			return nil, err
		}
	}

	value := traverseToKey(v.secret.Data, selector)
	if value == nil {
		return nil, &errBadSelector{fmt.Errorf("no value at path %q for key %q", v.path, selector)}
	}

	return newRetrieved(value, watchForUpdateFn), nil
}

func (v *vaultSession) RetrieveEnd(context.Context) error {
	return nil
}

func (v *vaultSession) Close(context.Context) error {
	close(v.doneCh)

	// Vault doesn't have a close for its client, close is completed.
	return nil
}

func newSession(client *api.Client, path string, logger *zap.Logger, pollInterval time.Duration) (*vaultSession, error) {
	if pollInterval <= 0 {
		return nil, errInvalidPollInterval
	}

	return &vaultSession{
		logger:       logger,
		client:       client,
		path:         path,
		pollInterval: pollInterval,
		doneCh:       make(chan struct{}),
	}, nil
}

func (v *vaultSession) readSecret() error {
	secret, err := v.client.Logical().Read(v.path)
	if err != nil {
		return &errClientRead{err}
	}

	// Invalid path does not return error but a nil secret.
	if secret == nil {
		return &errNilSecret{fmt.Errorf("no secret found at %q", v.path)}
	}

	// Incorrect path for v2 return nil data and warnings.
	if secret.Data == nil {
		return &errNilSecretData{fmt.Errorf("no data at %q warnings: %v", v.path, secret.Warnings)}
	}

	v.secret = secret
	return nil
}

func (v *vaultSession) buildWatcherFn() (func() error, error) {
	switch {
	case v.secret.Renewable:
		// Dynamic secret supporting renewal.
		return v.buildLifetimeWatcher()
	case v.secret.LeaseDuration > 0:
		// Version 1 lease: re-fetch it periodically.
		return v.buildV1LeaseWatcher()
	default:
		// Not a dynamic secret the best that can be done is polling.
		return v.buildPollingWatcher()
	}
}

func (v *vaultSession) buildLifetimeWatcher() (func() error, error) {
	vaultWatcher, err := v.client.NewLifetimeWatcher(&api.RenewerInput{
		Secret: v.secret,
	})
	if err != nil {
		return nil, err
	}

	watcherFn := func() error {
		go vaultWatcher.Start()
		defer vaultWatcher.Stop()

		for {
			select {
			case <-vaultWatcher.RenewCh():
				v.logger.Debug("vault secret renewed", zap.String("path", v.path))
			case err := <-vaultWatcher.DoneCh():
				// Renewal stopped, error or not the client needs to re-fetch the configuration.
				if err == nil {
					return configsource.ErrValueUpdated
				}
				return err
			case <-v.doneCh:
				return configsource.ErrSessionClosed
			}
		}
	}

	return watcherFn, nil
}

// buildV1LeaseWatcher builds a watcher function that takes the TTL given
// by Vault and triggers the re-fetch of the secret when half of the TTl
// has passed. In principle, this could be changed to actually check if the
// values of the secret were actually changed or not.
func (v *vaultSession) buildV1LeaseWatcher() (func() error, error) {
	watcherFn := func() error {
		// The lease duration is a hint of time to re-fetch the values.
		// The SmartAgent waits for half ot the lease duration.
		updateWait := time.Duration(v.secret.LeaseDuration/2) * time.Second
		select {
		case <-time.After(updateWait):
			// This is triggering a re-fetch. In principle this could actually
			// check for changes in the values.
			return configsource.ErrValueUpdated
		case <-v.doneCh:
			return configsource.ErrSessionClosed
		}
	}

	return watcherFn, nil
}

// buildPollingWatcher builds a watcher function that monitors for changes on
// the v.secret metadata. In principle this could be done for the actual value of
// the retrieved keys. However, checking for metadata keeps this in sync with the
// SignalFx SmartAgent behavior.
func (v *vaultSession) buildPollingWatcher() (func() error, error) {
	// Use the same requirements as SignalFx Smart Agent to build a polling watcher for the secret:
	//
	// This secret is not renewable or on a lease.  If it has a
	// "metadata" field and has "/data/" in the vault path, then it is
	// probably a KV v2 secret.  In that case, we do a poll on the
	// secret's metadata to refresh it and notice if a new version is
	// added to the secret.
	mdValue := v.secret.Data["metadata"]
	if mdValue == nil || !strings.Contains(v.path, "/data/") {
		v.logger.Warn("Missing metadata to create polling watcher for vault config source", zap.String("path", v.path))
		return watcherNotSupported, nil
	}

	mdMap, ok := mdValue.(map[string]interface{})
	if !ok {
		v.logger.Warn("Metadata not in the expected format to create polling watcher for vault config source", zap.String("path", v.path))
		return watcherNotSupported, nil
	}

	originalVersion := v.extractVersionMetadata(mdMap, "created_time", "version")
	if originalVersion == nil {
		v.logger.Warn("Failed to extract version metadata to create to create polling watcher for vault config source", zap.String("path", v.path))
		return watcherNotSupported, nil
	}

	watcherFn := func() error {
		metadataPath := strings.Replace(v.path, "/data/", "/metadata/", 1)
		ticker := time.NewTicker(v.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metadataSecret, err := v.client.Logical().Read(metadataPath)
				if err != nil {
					// Docs are not clear about how to differentiate between temporary and permanent errors.
					// Assume that the configuration needs to be re-fetched.
					return fmt.Errorf("failed to read secret metadata at %q: %w", metadataPath, err)
				}

				if metadataSecret == nil || metadataSecret.Data == nil {
					return fmt.Errorf("no secret metadata found at %q", metadataPath)
				}

				const timestampKey = "updated_time"
				const versionKey = "current_version"
				latestVersion := v.extractVersionMetadata(metadataSecret.Data, timestampKey, versionKey)
				if latestVersion == nil {
					return fmt.Errorf("secret metadata is not in the expected format for keys %q and %q", timestampKey, versionKey)
				}

				// Per SmartAgent code this is enough to trigger an update but it is also possible to check if the
				// the valued of the retrieved keys was changed. The current criteria may trigger updates even for
				// addition of new keys to the secret.
				if originalVersion.Timestamp != latestVersion.Timestamp || originalVersion.Version != latestVersion.Version {
					return configsource.ErrValueUpdated
				}
			case <-v.doneCh:
				return configsource.ErrSessionClosed
			}
		}
	}

	return watcherFn, nil
}

type versionMetadata struct {
	Timestamp string
	Version   int64
}

func (v *vaultSession) extractVersionMetadata(metadataMap map[string]interface{}, timestampKey, versionKey string) *versionMetadata {
	timestamp, ok := metadataMap[timestampKey].(string)
	if !ok {
		v.logger.Warn("Missing or unexpected type for timestamp on the metadata map", zap.String("key", timestampKey))
		return nil
	}

	versionNumber, ok := metadataMap[versionKey].(json.Number)
	if !ok {
		v.logger.Warn("Missing or unexpected type for version on the metadata map", zap.String("key", versionKey))
		return nil
	}

	versionInt, err := versionNumber.Int64()
	if err != nil {
		v.logger.Warn("Failed to parse version number into an integer", zap.String("key", versionKey), zap.String("version_number", string(versionNumber)))
		return nil
	}

	return &versionMetadata{
		Timestamp: timestamp,
		Version:   versionInt,
	}
}

// Allows key to be dot-delimited to traverse nested maps.
func traverseToKey(data map[string]interface{}, key string) interface{} {
	parts := strings.Split(key, ".")

	for i := 0; ; i++ {
		partVal := data[parts[i]]
		if i == len(parts)-1 {
			return partVal
		}

		var ok bool
		data, ok = partVal.(map[string]interface{})
		if !ok {
			return nil
		}
	}
}

func watcherNotSupported() error {
	return configsource.ErrWatcherNotSupported
}

type retrieved struct {
	value            interface{}
	watchForUpdateFn func() error
}

var _ configsource.Retrieved = (*retrieved)(nil)

func (r *retrieved) Value() interface{} {
	return r.value
}

func (r *retrieved) WatchForUpdate() error {
	return r.watchForUpdateFn()
}

func newRetrieved(value interface{}, watchForUpdateFn func() error) *retrieved {
	return &retrieved{
		value,
		watchForUpdateFn,
	}
}
