// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	wantCfg := &testConfig{}
	require.NoError(t, cm.Unmarshal(wantCfg, WithScalarUnmarshaler()))
	require.NoError(t, cm.Marshal(wantCfg, WithScalarMarshaler()))

	conf := confmap.New()
	cfg := &testConfig{
		Tma:     textMarshalerAlias("test"),
		Ntma:    nonTextMarshalerAlias("test"),
		Implint: wrapperType[int]{inner: 1},
		Implstr: wrapperType[string]{inner: "test"},
		Impltms: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}},
	}

	require.NoError(t, conf.Unmarshal(cfg))
	require.Equal(t, wantCfg, cfg)
}
