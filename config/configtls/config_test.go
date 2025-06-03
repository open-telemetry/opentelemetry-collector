package configtls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate_CertKeyPresence(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name:      "no cert or key",
			config:    Config{},
			expectErr: false,
		},
		{
			name:      "only CertFile",
			config:    Config{CertFile: "cert.pem"},
			expectErr: true,
		},
		{
			name:      "only KeyFile",
			config:    Config{KeyFile: "key.pem"},
			expectErr: true,
		},
		{
			name:      "CertFile and KeyFile",
			config:    Config{CertFile: "cert.pem", KeyFile: "Key.pem"},
			expectErr: false,
		},
		{
			name:      "CertPem and KeyPem",
			config:    Config{CertPem: "cert", KeyPem: "key"},
			expectErr: false,
		},
		{
			name:      "CertFile and KeyPem(mixed)",
			config:    Config{CertFile: "cert.pem", KeyPem: "key"},
			expectErr: false,
		},
		{
			name:      "CertPem only",
			config:    Config{CertPem: "cert"},
			expectErr: true,
		},
		{
			name:      "KeyPem only",
			config:    Config{KeyPem: "key"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
