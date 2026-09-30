// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"

	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
)

type Config struct {
	Message     string `mapstructure:"message"`
	Error       string `mapstructure:"error"`
	Unsupported bool   `mapstructure:"unsupported"`
}

type dummyPlugin struct{}

func runAction(action string, rawCfg map[string]any, unsupportedErr error) error {
	var cfg Config
	if err := mapstructure.Decode(rawCfg, &cfg); err != nil {
		return fmt.Errorf("failed to decode dummy plugin configuration: %w", err)
	}
	if cfg.Unsupported {
		return unsupportedErr
	}
	if cfg.Error != "" {
		return errors.New(cfg.Error)
	}
	fmt.Printf("%s:%s\n", action, cfg.Message)
	return nil
}

func (d *dummyPlugin) PreGenerate(config map[string]any) error {
	return runAction("pre-generate", config, ocbplugin.ErrUnsupportedActionPreGenerate)
}

func (d *dummyPlugin) PostGenerate(config map[string]any) error {
	return runAction("post-generate", config, ocbplugin.ErrUnsupportedActionPostGenerate)
}

func (d *dummyPlugin) PreBuild(config map[string]any) error {
	return runAction("pre-build", config, ocbplugin.ErrUnsupportedActionPreBuild)
}

func (d *dummyPlugin) PostBuild(config map[string]any) error {
	return runAction("post-build", config, ocbplugin.ErrUnsupportedActionPostBuild)
}

func (d *dummyPlugin) MinOCBVersion() string {
	return "0.151.0"
}

func main() {
	ocbplugin.RunPlugin(&dummyPlugin{})
}
