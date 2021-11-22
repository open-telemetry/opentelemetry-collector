package env

import (
	"context"
	"os"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

type configSource struct {
}

func (c configSource) RetrieveValue(
	ctx context.Context, onChange func(*configmapprovider.ChangeEvent), selector string, paramsConfigMap *config.Map,
) (configmapprovider.RetrievedValue, error) {
	return &retrieved{selector: selector, paramsConfigMap: paramsConfigMap}, nil
}

func (c configSource) Shutdown(ctx context.Context) error {
	return nil
}

type retrieved struct {
	selector        string
	paramsConfigMap *config.Map
}

func (r retrieved) Get(ctx context.Context) (interface{}, error) {
	return os.Getenv(r.selector), nil
}

func (r retrieved) Close(ctx context.Context) error {
	return nil
}
