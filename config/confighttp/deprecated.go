// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
)

type renamedField struct {
	old string
	new string
}

func (f *renamedField) Log(logger *zap.Logger) {
	// Note: This intentionally embeds the field names in the message instead of zap fields for readability
	logger.Warn(fmt.Sprintf("Use of deprecated configuration option `%s`, use `%s` instead.", f.old, f.new))
}

// collectLegacyKeepaliveWarnings returns a warning for each deprecated field that
// is present in conf. It returns an error if any deprecated field is used together
// with the new "keepalive" section.
func collectLegacyKeepaliveWarnings(conf *confmap.Conf, fields []renamedField, configName string) ([]renamedField, error) {
	var warnings []renamedField
	for _, field := range fields {
		if conf.IsSet(field.old) {
			warnings = append(warnings, field)
		}
	}
	if len(warnings) > 0 && conf.IsSet("keepalive") {
		return nil, fmt.Errorf("%s: cannot use legacy fields and new 'keepalive' section", configName)
	}
	return warnings, nil
}

// legacyClientKeepaliveFields holds the pre-keepalive client fields accepted during Unmarshal.
type legacyClientKeepaliveFields struct {
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host,omitempty"`
	DisableKeepAlives   bool          `mapstructure:"disable_keep_alives,omitempty"`
}

var clientKeepaliveDeprecations = []renamedField{
	{"idle_conn_timeout", "keepalive::idle_conn_timeout"},
	{"max_idle_conns", "keepalive::max_idle_conns"},
	{"disable_keep_alives", "keepalive::enabled"},
	{"max_idle_conns_per_host", "keepalive::max_idle_conns_per_host"},
}

// unmarshalClientDeprecated unmarshals conf into cc, translating any deprecated fields into their stable
// counterparts.
func unmarshalClientDeprecated(cc *ClientConfig, conf *confmap.Conf) error {
	type clientConfigFields ClientConfig
	type legacyConfig struct {
		clientConfigFields          `mapstructure:",squash"`
		legacyClientKeepaliveFields `mapstructure:",squash"`
	}

	defaultTransport := http.DefaultTransport.(*http.Transport)
	cfg := legacyConfig{
		clientConfigFields: clientConfigFields(*cc),
		legacyClientKeepaliveFields: legacyClientKeepaliveFields{
			MaxIdleConns:    defaultTransport.MaxIdleConns,
			IdleConnTimeout: defaultTransport.IdleConnTimeout,
		},
	}
	if err := conf.Unmarshal(&cfg, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	warnings, err := collectLegacyKeepaliveWarnings(conf, clientKeepaliveDeprecations, "confighttp.ClientConfig")
	if err != nil {
		return err
	}
	applyLegacyClientKeepalive(conf, &cfg.Keepalive, cfg.legacyClientKeepaliveFields)

	*cc = ClientConfig(cfg.clientConfigFields)
	cc.unmarshalWarnings = warnings
	return nil
}

// applyLegacyClientKeepalive maps legacy keepalive fields onto the new Keepalive optional.
func applyLegacyClientKeepalive(conf *confmap.Conf, keepalive *configoptional.Optional[KeepaliveClientConfig], legacy legacyClientKeepaliveFields) {
	if legacy.DisableKeepAlives {
		*keepalive = configoptional.None[KeepaliveClientConfig]()
		return
	}
	// If no legacy fields are present, defer entirely to whatever the new `keepalive`
	// section (or its absence) already resolved to. Using both legacy fields and the
	// new section simultaneously is rejected by collectLegacyKeepaliveWarnings before
	// this function is called, so that case cannot arise here.
	if !conf.IsSet("idle_conn_timeout") && !conf.IsSet("max_idle_conns") && !conf.IsSet("max_idle_conns_per_host") {
		return
	}
	if !keepalive.HasValue() {
		*keepalive = configoptional.Some(KeepaliveClientConfig{})
	}
	if conf.IsSet("idle_conn_timeout") {
		keepalive.Get().IdleConnTimeout = legacy.IdleConnTimeout
	}
	if conf.IsSet("max_idle_conns") {
		keepalive.Get().MaxIdleConns = legacy.MaxIdleConns
	}
	if conf.IsSet("max_idle_conns_per_host") {
		keepalive.Get().MaxIdleConnsPerHost = legacy.MaxIdleConnsPerHost
	}
}

// legacyServerKeepaliveFields holds the pre-keepalive server fields accepted during Unmarshal.
type legacyServerKeepaliveFields struct {
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	KeepAlivesEnabled bool          `mapstructure:"keep_alives_enabled,omitempty"`
}

var serverKeepaliveDeprecations = []renamedField{
	{"idle_timeout", "keepalive::idle_timeout"},
	{"keep_alives_enabled", "keepalive::enabled"},
}

// unmarshalServerDeprecated unmarshals conf into sc, translating any deprecated fields into their stable
// counterparts.
func unmarshalServerDeprecated(sc *ServerConfig, conf *confmap.Conf) error {
	type serverConfigFields ServerConfig
	type legacyConfig struct {
		serverConfigFields          `mapstructure:",squash"`
		legacyServerKeepaliveFields `mapstructure:",squash"`
	}

	cfg := legacyConfig{
		serverConfigFields: serverConfigFields(*sc),
		legacyServerKeepaliveFields: legacyServerKeepaliveFields{
			IdleTimeout:       1 * time.Minute,
			KeepAlivesEnabled: true,
		},
	}
	if err := conf.Unmarshal(&cfg, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	warnings, err := collectLegacyKeepaliveWarnings(conf, serverKeepaliveDeprecations, "confighttp.ServerConfig")
	if err != nil {
		return err
	}
	applyLegacyServerKeepalive(conf, &cfg.Keepalive, cfg.legacyServerKeepaliveFields)

	*sc = ServerConfig(cfg.serverConfigFields)
	sc.renamedFields = warnings
	return nil
}

// applyLegacyServerKeepalive maps legacy keepalive fields onto the new Keepalive optional.
func applyLegacyServerKeepalive(conf *confmap.Conf, keepalive *configoptional.Optional[KeepaliveServerConfig], legacy legacyServerKeepaliveFields) {
	if !legacy.KeepAlivesEnabled {
		*keepalive = configoptional.None[KeepaliveServerConfig]()
		return
	}
	// If no legacy fields are present, defer entirely to whatever the new `keepalive`
	// section (or its absence) already resolved to. Using both legacy fields and the
	// new section simultaneously is rejected by collectLegacyKeepaliveWarnings before
	// this function is called, so that case cannot arise here.
	if !conf.IsSet("idle_timeout") {
		return
	}
	if !keepalive.HasValue() {
		*keepalive = configoptional.Some(KeepaliveServerConfig{})
	}
	keepalive.Get().IdleTimeout = legacy.IdleTimeout
}
