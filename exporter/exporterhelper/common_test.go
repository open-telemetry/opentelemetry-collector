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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
)

var defaultExporterCfg = &configmodels.ExporterSettings{
	TypeVal: "test",
	NameVal: "test",
}

func TestBaseExporter(t *testing.T) {
	be := newBaseExporter(defaultExporterCfg)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	be := newBaseExporter(
		defaultExporterCfg,
		WithStart(func(ctx context.Context, host component.Host) error { return errors.New("my error") }),
		WithShutdown(func(ctx context.Context) error { return errors.New("my error") }))
	require.Error(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Error(t, be.Shutdown(context.Background()))
}
