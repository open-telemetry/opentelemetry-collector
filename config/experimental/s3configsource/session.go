package s3configsource

import (
	"context"

	"github.com/knadh/koanf"
	splunkprovider "github.com/signalfx/splunk-otel-collector"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.uber.org/zap"
)

type s3Session struct {
	logger         *zap.Logger
	internalConfig *koanf.Koanf
}

var _ configsource.Session = (*s3Session)(nil)

func (s *s3Session) Retrieve(_ context.Context, selector string, _ interface{}) (configsource.Retrieved, error) {
	return splunkprovider.NewRetrieved(s.internalConfig.String(selector), splunkprovider.WatcherNotSupported), nil
}

func (s *s3Session) RetrieveEnd(context.Context) error {
	return nil
}

func (s *s3Session) Close(context.Context) error {
	return nil
}

func newSession(logger *zap.Logger, internalConfig koanf.Koanf) (*s3Session, error) {
	return &s3Session{
		logger:         logger,
		internalConfig: &internalConfig,
	}, nil
}
