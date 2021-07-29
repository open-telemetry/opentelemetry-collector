package s3configsource

import (
	"bytes"
	"context"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.uber.org/zap"
)

type s3ConfigSource struct {
	logger         *zap.Logger
	internalConfig *configparser.Parser
}

var _ configsource.ConfigSource = (*s3ConfigSource)(nil)

func (s *s3ConfigSource) NewSession(context.Context) (configsource.Session, error) {
	return newSession(s.logger, *s.internalConfig)
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

	if cfg.VersionId == "" {
		_, err = downloader.Download(buf,
			&s3.GetObjectInput{
				Bucket: aws.String(cfg.Bucket),
				Key:    aws.String(cfg.Key),
			})
	} else {
		_, err = downloader.Download(buf,
			&s3.GetObjectInput{
				Bucket:    aws.String(cfg.Bucket),
				Key:       aws.String(cfg.Key),
				VersionId: aws.String(cfg.VersionId),
			})
	}

	if err != nil {
		return nil, err
	}

	data := buf.Bytes()
	ioReader := bytes.NewReader(data)
	internalConfig, err := configparser.NewParserFromBuffer(ioReader)

	if err != nil {
		log.Fatalf("error cloading config %v", err)
	}

	return &s3ConfigSource{
		logger:         logger,
		internalConfig: internalConfig,
	}, nil

}
