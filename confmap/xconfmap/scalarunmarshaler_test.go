// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalConfig(t *testing.T) {
	wantCfg := &testConfig{
		Tma:       textMarshalerAlias("test"),
		Ntma:      nonTextMarshalerAlias("test"),
		Implint:   wrapperType[int]{inner: 1},
		Implstr:   wrapperType[string]{inner: "test"},
		Impltms:   wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}},
		Recursive: wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}}},
	}

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := &testConfig{}
	require.NoError(t, cm.Unmarshal(cfg, WithScalarUnmarshaler()))

	require.Equal(t, wantCfg, cfg)
}
