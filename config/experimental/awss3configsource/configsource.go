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
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.uber.org/zap"
)

type s3ConfigSource struct {
	logger       *zap.Logger
	configParser *configparser.Parser
}

var _ configsource.ConfigSource = (*s3ConfigSource)(nil)

func (s *s3ConfigSource) NewSession(context.Context) (configsource.Session, error) {
	return newSession(s.logger, *s.configParser)
}

func newConfigSource(logger *zap.Logger, cfg *Config) (*s3ConfigSource, error) {

	buf := &aws.WriteAtBuffer{}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Region)},
	)

	if err != nil {
		return nil, err
	}

	downloader := s3manager.NewDownloader(sess)

	input := &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	}

	if cfg.VersionID != "" {
		input.VersionId = aws.String(cfg.VersionID)
	}

	_, err = downloader.Download(buf, input)

	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(buf.Bytes())
	configParser, err := configparser.NewParserFromBuffer(reader)

	if err != nil {
		return nil, err
	}

	return &s3ConfigSource{
		logger:       logger,
		configParser: configParser,
	}, nil

}
