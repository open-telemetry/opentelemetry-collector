// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"net/http"

	"go.opentelemetry.io/collector/component/componenttest"
)

func ExampleServerConfig() {
	settings := NewDefaultServerConfig()
	settings.Endpoint = "localhost:443"

	// Typically obtained as an argument of Component.Start()
	host := componenttest.NewNopHost()

	s, err := settings.ToServer(
		context.Background(),
		host.GetExtensions(),
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err != nil {
		panic(err)
	}

	l, err := settings.ToListener(context.Background())
	if err != nil {
		panic(err)
	}
	if err = s.Serve(l); err != nil {
		panic(err)
	}
}
