// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurvePreferencesRequireFIPSBuildTag(t *testing.T) {
	for _, curve := range []string{"X25519", "X25519MLKEM768", "SecP256r1MLKEM768", "SecP384r1MLKEM1024"} {
		t.Run(curve, func(t *testing.T) {
			cfg := ClientConfig{Config: Config{CurvePreferences: []string{curve}}}
			require.ErrorContains(t, cfg.Validate(), "invalid curve type: "+curve)
			_, err := cfg.LoadTLSConfig(context.Background())
			require.ErrorContains(t, err, "invalid curve type: "+curve)
		})
	}

	cfg := ClientConfig{}
	require.NoError(t, cfg.Validate())
	tlsCfg, err := cfg.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521}, tlsCfg.CurvePreferences)
}
