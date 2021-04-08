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
	"time"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/internal/configsource"
)

type vaultConfigSource struct {
	client *api.Client
	path   string
}

var _ configsource.ConfigSource = (*vaultConfigSource)(nil)

func (v *vaultConfigSource) NewSession(context.Context) (configsource.Session, error) {
	// TODO: Logger and poll interval should not be hard coded here but come from factory creating the config source.
	return newSession(v.client, v.path, zap.NewNop(), 2*time.Second)
}

func newConfigSource(address, token, path string) (*vaultConfigSource, error) {
	// Client doesn't connect on creation and can't be closed. Keeping the same instance
	// for all sessions is ok.
	client, err := api.NewClient(&api.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	client.SetToken(token)

	return &vaultConfigSource{
		client: client,
		path:   path,
	}, nil
}
