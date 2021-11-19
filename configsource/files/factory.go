package files

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/internal/internalinterface"
)

type factory struct {
	internalinterface.BaseInternal
}

func (f factory) Type() config.Type {
	return "files"
}

func (f factory) CreateDefaultConfig() config.ConfigSource {
	return &Config{
		ConfigSourceSettings: config.NewConfigSourceSettings(config.NewComponentID(f.Type())),
	}
}

func (f factory) CreateConfigSource(
	ctx context.Context, set component.ConfigSourceCreateSettings, cfg config.ConfigSource,
) (component.ConfigSource, error) {
	return configmapprovider.NewFile(cfg.(*Config).Name), nil
}

func NewFactory() component.ConfigSourceFactory {
	return &factory{}
}
