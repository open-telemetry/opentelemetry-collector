// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This is an example of a provider that calls a provided WatcherFunc to update configuration dynamically every second.
// The example is useful for implementing Providers of configuration that changes over time.
type UpdatingProvider struct{}

func (p UpdatingProvider) getCurrentConfig(_ string) any {
	return "hello"
}

func (p UpdatingProvider) Retrieve(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error) {
	ticker := time.NewTicker(1 * time.Second)
	stop := make(chan bool, 1)

	retrieved, err := NewRetrieved(p.getCurrentConfig(uri), WithRetrievedClose(func(_ context.Context) error {
		// the retriever should call this function when it no longer wants config updates
		ticker.Stop()
		stop <- true
		return nil
	}))
	if err != nil {
		return nil, err
	}
	// it's necessary to start a go function that can notify the caller of changes asynchronously
	go func() {
		for {
			select {
			case <-ctx.Done():
				// if the context is closed, then we should stop sending updates
				ticker.Stop()
				return
			case <-stop:
				// closeFunc was called, so stop updating the watcher
				return
			case <-ticker.C:
				// the configuration has "changed". Notify the watcher that a new config is available
				// the watcher is expected to call Provider.Retrieve again to get the update
				// note that the collector calls closeFunc before calling Retrieve for the second time,
				// so these go functions don't accumulate indefinitely. (see otelcol/collector.go, Collector.reloadConfiguration)
				watcher(&ChangeEvent{})
			}
		}
	}()

	return retrieved, nil
}

func ExampleProvider() {
	provider := UpdatingProvider{}

	receivedNotification := make(chan bool)

	watcherFunc := func(_ *ChangeEvent) {
		fmt.Println("received notification of new config")
		receivedNotification <- true
	}

	retrieved, err := provider.Retrieve(context.Background(), "example", watcherFunc)
	if err != nil {
		fmt.Println("received an error")
	} else {
		fmt.Printf("received: %s\n", retrieved.rawConf)
	}

	// after one second, we should receive a notification that config has changed
	<-receivedNotification
	// signal that we no longer want updates
	retrieved.Close(context.Background())

	// Output:
	// received: hello
	// received notification of new config
}

func TestNewRetrieved(t *testing.T) {
	ret, err := NewRetrieved(nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrieved(nil, WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedUnsupportedType(t *testing.T) {
	_, err := NewRetrieved(errors.New("my error"))
	require.Error(t, err)
}

func TestNewRetrievedFromYAML(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte{})
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrievedFromYAML([]byte{}, WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLInvalidYAMLBytes(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("[invalid:,"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	require.EqualError(t, err,
		"retrieved value (type=string) cannot be used as a Conf: assuming string type since contents are not valid YAML: yaml: line 1: did not find expected node content",
	)

	str, err := ret.AsString()
	require.NoError(t, err)
	assert.Equal(t, "[invalid:,", str)

	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.Equal(t, "[invalid:,", raw)
}

func TestNewRetrievedFromYAMLInvalidAsMap(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("string"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	require.EqualError(t, err, "retrieved value (type=string) cannot be used as a Conf")

	str, err := ret.AsString()
	require.NoError(t, err)
	assert.Equal(t, "string", str)
}

func TestNewRetrievedFromYAMLString(t *testing.T) {
	tests := []struct {
		yaml       string
		value      any
		altStrRepr string
		strReprErr string
	}{
		{
			yaml:  "string",
			value: "string",
		},
		{
			yaml:       "\"string\"",
			value:      "\"string\"",
			altStrRepr: "\"string\"",
		},
		{
			yaml:  "123",
			value: 123,
		},
		{
			yaml:  "2023-03-20T03:17:55.432328Z",
			value: time.Date(2023, 3, 20, 3, 17, 55, 432328000, time.UTC),
		},
		{
			yaml:  "true",
			value: true,
		},
		{
			yaml:  "0123",
			value: 0o123,
		},
		{
			yaml:  "0x123",
			value: 0x123,
		},
		{
			yaml:  "0b101",
			value: 0b101,
		},
		{
			yaml:  "0.123",
			value: 0.123,
		},
		{
			yaml:  "{key: value}",
			value: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.yaml, func(t *testing.T) {
			ret, err := NewRetrievedFromYAML([]byte(tt.yaml))
			require.NoError(t, err)

			raw, err := ret.AsRaw()
			require.NoError(t, err)
			assert.Equal(t, tt.value, raw)

			str, err := ret.AsString()
			if tt.strReprErr != "" {
				assert.ErrorContains(t, err, tt.strReprErr)
				return
			}
			require.NoError(t, err)

			if tt.altStrRepr != "" {
				assert.Equal(t, tt.altStrRepr, str)
			} else {
				assert.Equal(t, tt.yaml, str)
			}
		})
	}
}
