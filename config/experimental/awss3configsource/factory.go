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

package awss3configsource

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/experimental/configsource"

	splunkprovider "github.com/signalfx/splunk-otel-collector"
)

const (
	typeStr = "aws_s3"
)

type (
	errMissingRegion struct{ error }
	errMissingBucket struct{ error }
	errMissingKey    struct{ error }
)

type s3Factory struct{}

func (s *s3Factory) Type() config.Type {
	return typeStr
}

func (s *s3Factory) CreateDefaultConfig() splunkprovider.ConfigSettings {
	return &Config{
		Settings: splunkprovider.NewSettings(typeStr),
	}
}

func (s *s3Factory) CreateConfigSource(_ context.Context, params splunkprovider.CreateParams, cfg splunkprovider.ConfigSettings) (configsource.ConfigSource, error) {
	s3Config := cfg.(*Config)

	if s3Config.Region == "" {
		return nil, &errMissingRegion{errors.New("no s3 region specified")}
	}

	if s3Config.Bucket == "" {
		return nil, &errMissingBucket{errors.New("no s3 bucket specified")}
	}

	if s3Config.Key == "" {
		return nil, &errMissingKey{errors.New("no s3 key specified")}
	}

	return newConfigSource(params.Logger, s3Config)
}

func NewFactory() splunkprovider.Factory {
	return &s3Factory{}
}
