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

package vaultconfigsource

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/experimental/configsource"
)

const (
	// The "type" of Vault config sources in configuration.
	typeStr = "vault"

	defaultPollInterval = 1 * time.Minute
)

// Private error types to help with testability.
type (
	errMissingEndpoint         struct{ error }
	errInvalidEndpoint         struct{ error }
	errMissingToken            struct{ error }
	errMissingPath             struct{ error }
	errNonPositivePollInterval struct{ error }
)

type vaultFactory struct{}

func (v *vaultFactory) Type() config.Type {
	return typeStr
}

func (v *vaultFactory) CreateDefaultConfig() configsource.ConfigSettings {
	return &Config{
		Settings:     configsource.NewSettings(typeStr),
		PollInterval: defaultPollInterval,
	}
}

func (v *vaultFactory) CreateConfigSource(_ context.Context, params configsource.CreateParams, cfg configsource.ConfigSettings) (configsource.ConfigSource, error) {
	vaultCfg := cfg.(*Config)

	if vaultCfg.Endpoint == "" {
		return nil, &errMissingEndpoint{errors.New("cannot connect to vault with an empty endpoint")}
	}

	if _, err := url.ParseRequestURI(vaultCfg.Endpoint); err != nil {
		return nil, &errInvalidEndpoint{fmt.Errorf("invalid endpoint %q: %w", vaultCfg.Endpoint, err)}
	}

	if vaultCfg.Token == "" {
		return nil, &errMissingToken{errors.New("cannot connect to vault with an empty token")}
	}

	if vaultCfg.Path == "" {
		return nil, &errMissingPath{errors.New("cannot connect to vault with an empty path")}
	}

	if vaultCfg.PollInterval <= 0 {
		return nil, &errNonPositivePollInterval{errors.New("poll_interval must to be positive")}
	}

	return newConfigSource(params.Logger, vaultCfg)
}

func NewFactory() configsource.Factory {
	return &vaultFactory{}
}
