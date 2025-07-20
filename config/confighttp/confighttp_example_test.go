// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"net/http"

	"go.opentelemetry.io/collector/component/componenttest"
)

func ExampleServerConfig() {
	settings := NewDefaultServerConfig()
	settings.Endpoint = "localhost:443"

	s, err := settings.ToServer(
		componenttest.NewNopHost(),
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err != nil {
		panic(err)
	}

	l, err := settings.ToListener()
	if err != nil {
		panic(err)
	}
	if err = s.Serve(l); err != nil {
		panic(err)
	}
}
