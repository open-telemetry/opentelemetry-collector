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

	"go.opentelemetry.io/collector/experimental/configsource"
)

type vaultConfigSource struct {
	logger       *zap.Logger
	client       *api.Client
	path         string
	pollInterval time.Duration
}

var _ configsource.ConfigSource = (*vaultConfigSource)(nil)

func (v *vaultConfigSource) NewSession(context.Context) (configsource.Session, error) {
	return newSession(v.client, v.path, v.logger, v.pollInterval)
}

func newConfigSource(logger *zap.Logger, cfg *Config) (*vaultConfigSource, error) {
	// Client doesn't connect on creation and can't be closed. Keeping the same instance
	// for all sessions is ok.
	client, err := api.NewClient(&api.Config{
		Address: cfg.Endpoint,
	})
	if err != nil {
		return nil, err
	}

	client.SetToken(cfg.Token)

	return &vaultConfigSource{
		logger:       logger,
		client:       client,
		path:         cfg.Path,
		pollInterval: cfg.PollInterval,
	}, nil
}
