// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package exporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

var (
	defaultExporterCfg  = config.NewExporterSettings(config.NewID(typeStr))
	exporterTag, _      = tag.NewKey("exporter")
	defaultExporterTags = []tag.Tag{
		{Key: exporterTag, Value: "test"},
	}
)

func TestBaseExporter(t *testing.T) {
	be := newBaseExporter(&defaultExporterCfg, componenttest.NewNopExporterCreateSettings(), fromOptions())
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be := newBaseExporter(
		&defaultExporterCfg,
		componenttest.NewNopExporterCreateSettings(),
		fromOptions(
			WithStart(func(ctx context.Context, host component.Host) error { return want }),
			WithShutdown(func(ctx context.Context) error { return want }),
			WithTimeout(DefaultTimeoutSettings())),
	)
	require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Equal(t, want, be.Shutdown(context.Background()))
}

func checkStatus(t *testing.T, sd sdktrace.ReadOnlySpan, err error) {
	if err != nil {
		require.Equal(t, codes.Error, sd.Status().Code, "SpanData %v", sd)
		require.Equal(t, err.Error(), sd.Status().Description, "SpanData %v", sd)
	} else {
		require.Equal(t, codes.Unset, sd.Status().Code, "SpanData %v", sd)
	}
}
