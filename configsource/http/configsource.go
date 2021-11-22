package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

type configSource struct {
	scheme string
}

func (c configSource) Retrieve(
	ctx context.Context, onChange func(*configmapprovider.ChangeEvent), selector string, paramsConfigMap *config.Map,
) (configmapprovider.RetrievedValue, error) {
	return &retrieved{scheme: c.scheme, selector: selector, paramsConfigMap: paramsConfigMap}, nil
}

func (c configSource) Shutdown(ctx context.Context) error {
	return nil
}

type retrieved struct {
	scheme          string
	selector        string
	paramsConfigMap *config.Map
}

func (r retrieved) Get(ctx context.Context) (interface{}, error) {

	// TODO: instead of passing scheme, selector and paramsConfigMap pass the entire
	// invocation string and avoid re-creating the URL here.

	urlStr := r.scheme + ":" + r.selector

	if r.paramsConfigMap != nil {
		queryVals := url.Values{}
		for k, v := range r.paramsConfigMap.ToStringMap() {
			str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid value for paramter %q", k)
			}
			queryVals.Set(k, str)
		}

		query := queryVals.Encode()
		if query != "" {
			urlStr = urlStr + "?" + query
		}
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("cannot load config from %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	// TODO: handle HTTP response codes.

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot load config from %s: %w", urlStr, err)
	}

	return bytes, nil
}

func (r retrieved) Close(ctx context.Context) error {
	return nil
}
