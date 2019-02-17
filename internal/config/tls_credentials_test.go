// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"reflect"
	"testing"
)

func TestTLSConfigByParsing(t *testing.T) {
	configYAML := []byte(`
receivers:
  opencensus:
    tls_credentials:
      cert_file: "foobar.crt"
      key_file: "foobar.key"
  `)

	cfg, err := ParseOCAgentConfig(configYAML)
	if err != nil {
		t.Fatalf("Failed to parse OCAgent config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Returned nil while parsing config")
	}

	tlsCreds := cfg.OpenCensusReceiverTLSServerCredentials()
	if tlsCreds == nil {
		t.Error("Surprisingly turned out nil TLS credentials")
	}

	if !tlsCreds.nonEmpty() {
		t.Error("nonEmpty returned false")
	}

	want := &TLSCredentials{
		CertFile: "foobar.crt",
		KeyFile:  "foobar.key",
	}

	if !reflect.DeepEqual(tlsCreds, want) {
		t.Errorf("Got:  %+v\nWant: %+v", cfg, want)
	}
}

func TestTLSConfigDereferencing(t *testing.T) {
	var nilConfig *Config
	if g := nilConfig.OpenCensusReceiverTLSServerCredentials(); g != nil {
		t.Errorf("Retrieved non-nil TLSServerCredentials: %+v\n", g)
	}

	if nilConfig.openCensusReceiverEnabled() {
		t.Error("Somehow OpenCensus receiver is enabled on a nil Config")
	}
}

func TestTLSCredentials_nonEmptyChecks(t *testing.T) {
	// TLSCredentials are considered "nonEmpty" if at least either
	// of "cert_file" or "key_file" are non-empty.
	combinations := []struct {
		config string
		want   bool
	}{
		{config: ``, want: false},
		{
			config: `
receivers:
  opencensus:
    tls_credentials:
      cert_file: "foo"
        `, want: true,
		},
		{
			config: `
receivers:
  opencensus:
    tls_credentials:
      key_file: "foo"
        `, want: true,
		},
		{
			config: `
receivers:
  opencensus:
    tls_credentials:
      key_file: ""
      cert_file: ""
        `, want: false,
		},
	}

	for i, tt := range combinations {
		cfg, err := ParseOCAgentConfig([]byte(tt.config))
		if err != nil {
			t.Errorf("#%d: unexpected parsing error: %v", i, err)
		}
		tlsCreds := cfg.OpenCensusReceiverTLSServerCredentials()
		got, want := tlsCreds.nonEmpty(), tt.want
		if got != want {
			t.Errorf("#%d: got=%t want=%t\nConfig:\n%s", i, got, want, tt.config)
		}
	}
}
