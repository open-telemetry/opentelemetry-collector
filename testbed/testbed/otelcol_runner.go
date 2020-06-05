// Copyright 2020, OpenTelemetry Authors
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

package testbed

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service"
)

type OtelcolRunner interface {
	PrepareConfig(configStr string) (string, error)
	Start(args StartParams) (string, error)
	Stop() (bool, error)
}

type InProcessPipeline struct {
	logger    *zap.Logger
	factories config.Factories
	config    *configmodels.Config
	svc       *service.Application
	appDone   chan struct{}
}

func NewInProcessPipeline(factories config.Factories) *InProcessPipeline {
	return &InProcessPipeline{
		factories: factories,
	}
}

func (ipp *InProcessPipeline) PrepareConfig(configStr string) (string, error) {
	logger, err := configureLogger()
	if err != nil {
		return "", err
	}
	ipp.logger = logger
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(configStr))
	cfg, err := config.Load(v, ipp.factories)
	if err != nil {
		return "", err
	}
	err = config.ValidateConfig(cfg, zap.NewNop())
	if err != nil {
		return "", err
	}
	ipp.config = cfg
	return "", nil
}

func (ipp *InProcessPipeline) Start(args StartParams) (string, error) {
	params := service.Parameters{
		ApplicationStartInfo: service.ApplicationStartInfo{
			ExeName:  "otelcol",
			LongName: "InProcess Collector",
			Version:  version.Version,
			GitHash:  version.GitHash,
		},
		ConfigFactory: func(v *viper.Viper, factories config.Factories) (*configmodels.Config, error) {
			return ipp.config, nil
		},
		Factories: ipp.factories,
	}
	var err error
	ipp.svc, err = service.New(params)
	if err != nil {
		return "", err
	}

	ipp.appDone = make(chan struct{})
	go func() {
		defer close(ipp.appDone)
		appErr := ipp.svc.Start()
		if appErr != nil {
			err = appErr
		}
	}()

	for state := range ipp.svc.GetStateChannel() {
		switch state {
		case service.Starting:
			// NoOp
		case service.Running:
			return DefaultHost, err
		default:
			err = fmt.Errorf("unable to start, otelcol state is %d", state)
		}
	}
	return "", err
}

func (ipp *InProcessPipeline) Stop() (bool, error) {
	ipp.svc.SignalTestComplete()
	<-ipp.appDone
	return true, nil
}

func configureLogger() (*zap.Logger, error) {
	conf := zap.NewDevelopmentConfig()
	conf.Level.SetLevel(zapcore.InfoLevel)
	logger, err := conf.Build()
	return logger, err
}
