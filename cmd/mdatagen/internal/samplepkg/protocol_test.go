// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplepkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolUnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		initial Protocol
		want    Protocol
		wantErr string
	}{
		{
			name:  "http",
			input: "http",
			want:  Protocol(ProtocolHTTP),
		},
		{
			name:  "tcp",
			input: "tcp",
			want:  Protocol(ProtocolTCP),
		},
		{
			name:  "smtp",
			input: "smtp",
			want:  Protocol(ProtocolSMTP),
		},
		{
			name:  "ftp",
			input: "ftp",
			want:  Protocol(ProtocolFTP),
		},
		{
			name:    "invalid",
			input:   "invalid",
			initial: Protocol(ProtocolTCP),
			want:    Protocol(ProtocolTCP),
			wantErr: "unknown protocol: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			protocol := tt.initial
			err := protocol.UnmarshalText([]byte(tt.input))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, protocol)
		})
	}
}
