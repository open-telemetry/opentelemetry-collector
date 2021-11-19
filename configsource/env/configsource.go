package env

import (
	"context"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

type configSource struct {
}

func (c configSource) Retrieve(
	ctx context.Context, onChange func(*configmapprovider.ChangeEvent),
) (configmapprovider.RetrievedConfig, error) {
	return &retrieved{}, nil
}

func (c configSource) Shutdown(ctx context.Context) error {
	return nil
}

type retrieved struct {
}

func (r retrieved) Get(ctx context.Context) (*config.Map, error) {
	return config.NewMap(), nil
}

func (r retrieved) Close(ctx context.Context) error {
	return nil
}
