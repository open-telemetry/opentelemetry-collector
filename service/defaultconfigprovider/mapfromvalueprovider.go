package defaultconfigprovider

import (
	"bytes"
	"context"
	"fmt"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

type mapFromValueProvider struct {
	configSourceName string
	valueProvider    configmapprovider.ValueProvider
	selector         string
	paramsConfigMap  *config.Map
}

func (m mapFromValueProvider) Shutdown(ctx context.Context) error {
	return m.valueProvider.Shutdown(ctx)
}

func (m mapFromValueProvider) Retrieve(
	ctx context.Context, onChange func(*configmapprovider.ChangeEvent),
) (configmapprovider.RetrievedMap, error) {
	retrieved, err := m.valueProvider.Retrieve(ctx, onChange, m.selector, m.paramsConfigMap)
	if err != nil {
		return nil, err
	}
	return &mapFromValueRetrieved{retrieved: retrieved, configSourceName: m.configSourceName}, nil
}

type mapFromValueRetrieved struct {
	retrieved        configmapprovider.RetrievedValue
	configSourceName string
}

func (m mapFromValueRetrieved) Get(ctx context.Context) (cfgMap *config.Map, err error) {
	val, err := m.retrieved.Get(ctx)
	if err != nil {
		return nil, err
	}

	switch v := val.(type) {
	case string:
		cfgMap, err = config.NewMapFromBuffer(bytes.NewReader([]byte(v)))
	case []byte:
		cfgMap, err = config.NewMapFromBuffer(bytes.NewReader(v))
	case *config.Map:
		cfgMap = v
	default:
		err = fmt.Errorf("config source %q returned invalid data (must return a string, []byte or a config.Map", m.configSourceName)
	}

	return cfgMap, err
}

func (m mapFromValueRetrieved) Close(ctx context.Context) error {
	return m.retrieved.Close(ctx)
}
