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
	"time"

	"go.opentelemetry.io/collector/config/internal/configsource"
)

type Config struct {
	*configsource.Settings
	// Endpoint is the address of the Vault server, typically it is set via the
	// VAULT_ADDR environment variable for the Vault CLI.
	Endpoint string `mapstructure:"endpoint"`
	// Token is the token to be used to access the Vault server, typically is set
	// via the VAULT_TOKEN environment variable for the Vault CLI.
	Token string `mapstructure:"token"`
	// Path is the Vault path where the secret to be retrieved is located.
	Path string `mapstructure:"path"`
	// PollInterval is the interval in which the config source will check for
	// changes on the data on the given Vault path. This is only used for
	// non-dynamic secret stores. Defaults to 1 minute if not specified.
	PollInterval time.Duration `mapstructure:"poll_interval"`
}
