package http

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/internal/internalinterface"
)

type factory struct {
	internalinterface.BaseInternal
	scheme string
}

func (f factory) Type() config.Type {
	return config.Type(f.scheme)
}

func (f factory) CreateDefaultConfig() config.ConfigSource {
	return &Config{
		ConfigSourceSettings: config.NewConfigSourceSettings(config.NewComponentID(f.Type())),
	}
}

func (f factory) CreateConfigSource(ctx context.Context, set component.ConfigSourceCreateSettings, cfg config.ConfigSource) (configmapprovider.Provider, error) {
	return &configSource{scheme: f.scheme}, nil
}

func NewHTTPFactory() component.ConfigSourceFactory {
	return &factory{scheme: "http"}
}

func NewHTTPSFactory() component.ConfigSourceFactory {
	return &factory{scheme: "https"}
}
