// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkaexporter

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"

	"github.com/Shopify/sarama"

	"go.opentelemetry.io/collector/config/configtls"
)

// Authentication defines authentication.
type Authentication struct {
	PlainText *PlainTextConfig            `mapstructure:"plain_text"`
	SASL      *SASLConfig                 `mapstructure:"sasl"`
	TLS       *configtls.TLSClientSetting `mapstructure:"tls"`
	Kerberos  *KerberosConfig             `mapstructure:"kerberos"`
}

// PlainTextConfig defines plaintext authentication.
type PlainTextConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// SASLConfig defines the configuration for the SASL authentication.
type SASLConfig struct {
	// Username to be used on authentication
	Username string `mapstructure:"username"`
	// Password to be used on authentication
	Password string `mapstructure:"password"`
	// SASL Mechanism to be used, possible values are: (PLAIN, SCRAM-SHA-256 or SCRAM-SHA-512).
	Mechanism string `mapstructure:"mechanism"`
}

// KerberosConfig defines kereros configuration.
type KerberosConfig struct {
	ServiceName string `mapstructure:"service_name"`
	Realm       string `mapstructure:"realm"`
	UseKeyTab   bool   `mapstructure:"use_keytab"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password" json:"-"`
	ConfigPath  string `mapstructure:"config_file"`
	KeyTabPath  string `mapstructure:"keytab_file"`
}

// ConfigureAuthentication configures authentication in sarama.Config.
func ConfigureAuthentication(config Authentication, saramaConfig *sarama.Config) error {
	if config.PlainText != nil {
		configurePlaintext(*config.PlainText, saramaConfig)
	}
	if config.TLS != nil {
		if err := configureTLS(*config.TLS, saramaConfig); err != nil {
			return err
		}
	}
	if config.SASL != nil {
		if err := configureSASL(*config.SASL, saramaConfig); err != nil {
			return err
		}
	}

	if config.Kerberos != nil {
		configureKerberos(*config.Kerberos, saramaConfig)
	}
	return nil
}

func configurePlaintext(config PlainTextConfig, saramaConfig *sarama.Config) {
	saramaConfig.Net.SASL.Enable = true
	saramaConfig.Net.SASL.User = config.Username
	saramaConfig.Net.SASL.Password = config.Password
}

func configureSASL(config SASLConfig, saramaConfig *sarama.Config) error {

	if config.Username == "" {
		return fmt.Errorf("username have to be provided")
	}

	if config.Password == "" {
		return fmt.Errorf("password have to be provided")
	}

	saramaConfig.Net.SASL.Enable = true
	saramaConfig.Net.SASL.User = config.Username
	saramaConfig.Net.SASL.Password = config.Password

	switch config.Mechanism {
	case "SCRAM-SHA-512":
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha512.New} }
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	case "SCRAM-SHA-256":
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha256.New} }
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	case "PLAIN":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	default:
		return fmt.Errorf("invalid SASL Mechanism %q: can be either \"PLAIN\" , \"SCRAM-SHA-256\" or \"SCRAM-SHA-512\"", config.Mechanism)
	}

	return nil
}

func configureTLS(config configtls.TLSClientSetting, saramaConfig *sarama.Config) error {
	tlsConfig, err := config.LoadTLSConfig()
	if err != nil {
		return fmt.Errorf("error loading tls config: %w", err)
	}
	saramaConfig.Net.TLS.Enable = true
	saramaConfig.Net.TLS.Config = tlsConfig
	return nil
}

func configureKerberos(config KerberosConfig, saramaConfig *sarama.Config) {
	saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
	saramaConfig.Net.SASL.Enable = true
	if config.UseKeyTab {
		saramaConfig.Net.SASL.GSSAPI.KeyTabPath = config.KeyTabPath
		saramaConfig.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
	} else {
		saramaConfig.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
		saramaConfig.Net.SASL.GSSAPI.Password = config.Password
	}
	saramaConfig.Net.SASL.GSSAPI.KerberosConfigPath = config.ConfigPath
	saramaConfig.Net.SASL.GSSAPI.Username = config.Username
	saramaConfig.Net.SASL.GSSAPI.Realm = config.Realm
	saramaConfig.Net.SASL.GSSAPI.ServiceName = config.ServiceName
}
