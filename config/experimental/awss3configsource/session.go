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

	splunkprovider "github.com/signalfx/splunk-otel-collector"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.uber.org/zap"
)

type s3Session struct {
	logger       *zap.Logger
	configParser *configparser.Parser
}

var _ configsource.Session = (*s3Session)(nil)

func (s *s3Session) Retrieve(_ context.Context, selector string, _ interface{}) (configsource.Retrieved, error) {
	return splunkprovider.NewRetrieved(s.configParser.Get(selector).(string), splunkprovider.WatcherNotSupported), nil
}

func (s *s3Session) RetrieveEnd(context.Context) error {
	return nil
}

func (s *s3Session) Close(context.Context) error {
	return nil
}

func newSession(logger *zap.Logger, configParser configparser.Parser) (*s3Session, error) {
	return &s3Session{
		logger:       logger,
		configParser: &configParser,
	}, nil
}
