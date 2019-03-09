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

package factorytemplate

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/exporter/exportertest"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

func TestNewTraceReceiverFactory(t *testing.T) {
	type args struct {
		receiverType  string
		newDefaultCfg func() interface{}
		newReceiver   func(interface{}, consumer.TraceConsumer, *zap.Logger) (receiver.TraceReceiver, error)
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr error
	}{
		{
			name:    "empty receiverType",
			want:    nil,
			wantErr: ErrEmptyReciverType,
		},
		{
			name: "nil newDefaultCfg",
			args: args{
				receiverType: "friendlyReceiverTypeName",
			},
			want:    nil,
			wantErr: ErrNilNewDefaultCfg,
		},
		{
			name: "nil newReceiver",
			args: args{
				receiverType:  "friendlyReceiverTypeName",
				newDefaultCfg: newMockReceiverDefaultCfg,
			},
			want:    nil,
			wantErr: ErrNilNewReceiver,
		},
		{
			name: "happy path",
			args: args{
				receiverType:  "friendlyReceiverTypeName",
				newDefaultCfg: newMockReceiverDefaultCfg,
				newReceiver:   newMockTraceReceiver,
			},
			want: &traceReceiverFactory{
				factory: factory{
					receiverType:  "friendlyReceiverTypeName",
					newDefaultCfg: newMockReceiverDefaultCfg,
				},
				newReceiver: newMockTraceReceiver,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTraceReceiverFactory(tt.args.receiverType, tt.args.newDefaultCfg, tt.args.newReceiver)
			if err != tt.wantErr {
				t.Errorf("NewTraceReceiverFactory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && tt.want == nil {
				return
			}
			want := tt.want.(receiver.TraceReceiverFactory)
			if got.Type() != want.Type() {
				t.Errorf("NewTraceReceiverFactory() = %v, want %v", got, want)
			}
		})
	}
}

func TestNewMetricsReceiverFactory(t *testing.T) {
	type args struct {
		receiverType  string
		newDefaultCfg func() interface{}
		newReceiver   func(interface{}, consumer.MetricsConsumer, *zap.Logger) (receiver.MetricsReceiver, error)
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr error
	}{
		{
			name:    "empty receiverType",
			want:    nil,
			wantErr: ErrEmptyReciverType,
		},
		{
			name: "nil newDefaultCfg",
			args: args{
				receiverType: "friendlyReceiverTypeName",
			},
			want:    nil,
			wantErr: ErrNilNewDefaultCfg,
		},
		{
			name: "nil newReceiver",
			args: args{
				receiverType:  "friendlyReceiverTypeName",
				newDefaultCfg: newMockReceiverDefaultCfg,
			},
			want:    nil,
			wantErr: ErrNilNewReceiver,
		},
		{
			name: "happy path",
			args: args{
				receiverType:  "friendlyReceiverTypeName",
				newDefaultCfg: newMockReceiverDefaultCfg,
				newReceiver:   newMockMetricsReceiver,
			},
			want: &metricsReceiverFactory{
				factory: factory{
					receiverType:  "friendlyReceiverTypeName",
					newDefaultCfg: newMockReceiverDefaultCfg,
				},
				newReceiver: newMockMetricsReceiver,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMetricsReceiverFactory(tt.args.receiverType, tt.args.newDefaultCfg, tt.args.newReceiver)
			if err != tt.wantErr {
				t.Errorf("NewMetricsReceiverFactory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && tt.want == nil {
				return
			}
			want := tt.want.(receiver.MetricsReceiverFactory)
			if got.Type() != want.Type() {
				t.Errorf("NewMetricsReceiverFactory() = %v, want %v", got, want)
			}
		})
	}
}

func Test_traceReceiverFactory_NewFromViper(t *testing.T) {
	factories := map[string]receiver.TraceReceiverFactory{
		"mockreceiver":    checkedBuildReceiverFactory(t, "mockReceiver", newMockReceiverDefaultCfg, newMockTraceReceiver),
		"altmockreceiver": checkedBuildReceiverFactory(t, "altMockReceiver", altNewMockReceiverDefaultCfg, newMockTraceReceiver),
	}

	v := viper.New()
	v.SetConfigFile("./testdata/test-config.yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config file for test: %v", err)
	}

	const subTree = "configurations.test-receivers"
	v = v.Sub(subTree)
	if v == nil {
		t.Fatalf("failed to find expected sub-tree %q on the configuration", subTree)
	}

	var got []mockReceiverCfg
	for subKey := range v.AllSettings() {
		subCfg := v.Sub(subKey)
		if subCfg == nil {
			t.Fatalf("missing expected section %q", subKey)
		}

		factory, hasFactory := factories[subKey]
		if !hasFactory {
			// Try to get from the "type"
			receiverType := subCfg.GetString("type")
			if receiverType == "" {
				t.Fatalf("there is no factory for %q and the corresponding \"type\" entry is empty", subKey)
			}
			factory, hasFactory = factories[receiverType]
			if !hasFactory {
				t.Fatalf("there is no factory for type %q (see section %q)", receiverType, subKey)
			}
			subKey = receiverType
		}

		r, err := factory.NewFromViper(subCfg, exportertest.NewNopTraceExporter(), zap.NewNop())
		if err != nil {
			t.Fatalf("failed to create receiver from factory: %v", err)
		}
		got = append(got, *r.(*mockReceiver).config)
	}

	want := []mockReceiverCfg{
		{
			Address: "0.0.0.0",
			Port:    616,
		},
		{
			Address: "altMockReceiverAddress",
			Port:    123,
		},
		{
			Address: "1.2.3.4",
			Port:    333,
		},
	}

	sort.Slice(got, func(i, j int) bool {
		return got[i].Address < got[j].Address
	})
	sort.Slice(want, func(i, j int) bool {
		return want[i].Address < want[j].Address
	})

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, len(want) = %d", len(got), len(want))
	}

	for i := range got {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("NewFromViper() at index %d got = %v, want = %v", i, got[i], want[i])
		}
	}
}

func Test_traceReceiverFactory_NewFromViper_Errors(t *testing.T) {
	errNewReceiver := errors.New("failed to create receiver")
	type args struct {
		v    *viper.Viper
		next consumer.TraceConsumer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "nil next",
			args: args{
				v: viper.New(),
			},
			wantErr: ErrNilNext,
		},
		{
			name: "nil viper",
			args: args{
				next: exportertest.NewNopTraceExporter(),
			},
			wantErr: ErrNilViper,
		},
		{
			name: "err newReceiver",
			args: args{
				v:    viper.New(),
				next: exportertest.NewNopTraceExporter(),
			},
			wantErr: errNewReceiver,
		},
	}

	newTraceReceiverAlwaysFail := func(cfg interface{}, next consumer.TraceConsumer, logger *zap.Logger) (receiver.TraceReceiver, error) {
		return nil, errNewReceiver
	}
	factory, err := NewTraceReceiverFactory(
		"mockTraceReceiver",
		newMockReceiverDefaultCfg,
		newTraceReceiverAlwaysFail,
	)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.NewFromViper(tt.args.v, tt.args.next, zap.NewNop())
			if err != tt.wantErr {
				t.Errorf("NewFromViper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_metricsReceiverFactory_NewFromViper(t *testing.T) {
	factory, err := NewMetricsReceiverFactory(
		"mockMetricsReceiver",
		newMockReceiverDefaultCfg,
		newMockMetricsReceiver,
	)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	v := viper.New()
	r, err := factory.NewFromViper(v, exportertest.NewNopMetricsExporter(), zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create receiver from factory: %v", err)
	}
	got := *r.(*mockReceiver).config

	want := mockReceiverCfg{
		Address: "mockReceiverAddress",
		Port:    616,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewFromViper() = %v, want = %v", got, want)
	}
}

func Test_metricsReceiverFactory_NewFromViper_Errors(t *testing.T) {
	errNewReceiver := errors.New("failed to create receiver")
	type args struct {
		v    *viper.Viper
		next consumer.MetricsConsumer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "nil next",
			args: args{
				v: viper.New(),
			},
			wantErr: ErrNilNext,
		},
		{
			name: "nil viper",
			args: args{
				next: exportertest.NewNopMetricsExporter(),
			},
			wantErr: ErrNilViper,
		},
		{
			name: "err newReceiver",
			args: args{
				v:    viper.New(),
				next: exportertest.NewNopMetricsExporter(),
			},
			wantErr: errNewReceiver,
		},
	}

	newMetricsReceiverAlwaysFail := func(cfg interface{}, next consumer.MetricsConsumer, logger *zap.Logger) (receiver.MetricsReceiver, error) {
		return nil, errNewReceiver
	}

	factory, err := NewMetricsReceiverFactory(
		"mockMetricsReceiver",
		newMockReceiverDefaultCfg,
		newMetricsReceiverAlwaysFail,
	)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.NewFromViper(tt.args.v, tt.args.next, zap.NewNop())
			if err != tt.wantErr {
				t.Errorf("NewFromViper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_factory_DefaultConfig(t *testing.T) {
	factory, err := NewMetricsReceiverFactory(
		"mockMetricsReceiver",
		newMockReceiverDefaultCfg,
		newMockMetricsReceiver,
	)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}
	cfg := factory.DefaultConfig()
	if cfg == nil {
		t.Fatalf("should have returned a non-nil config")
	}
	if _, ok := cfg.(*mockReceiverCfg); !ok {
		t.Fatalf("type assertion for default config failed")
	}
}

func TestInvalidConfig(t *testing.T) {
	factory, err := NewMetricsReceiverFactory(
		"mockMetricsReceiver",
		newMockReceiverDefaultCfg,
		newMockMetricsReceiver,
	)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	v := viper.New()
	v.Set("address", "myaddress")
	v.Set("port", "not_a_number")

	ne := exportertest.NewNopMetricsExporter()
	_, err = factory.NewFromViper(v, ne, zap.NewNop())
	if err == nil {
		t.Fatal("NewFromViper() got nil error for invalid config")
	}
}

func Examplefactory_DefaultConfig() {
	templateArgs := []struct {
		receiverType  string
		newDefaultCfg func() interface{}
		newReceiver   func(interface{}, consumer.TraceConsumer, *zap.Logger) (receiver.TraceReceiver, error)
	}{
		{
			receiverType:  "mockReceiver",
			newDefaultCfg: newMockReceiverDefaultCfg,
			newReceiver:   newMockTraceReceiver,
		},
		{
			receiverType:  "altMockReceiver",
			newDefaultCfg: altNewMockReceiverDefaultCfg,
			newReceiver:   newMockTraceReceiver,
		},
	}

	var factories []receiver.TraceReceiverFactory
	for _, args := range templateArgs {
		factory, _ := NewTraceReceiverFactory(args.receiverType, args.newDefaultCfg, args.newReceiver)
		factories = append(factories, factory)
	}

	v := viper.New()
	for _, factory := range factories {
		v.SetDefault("receivers."+factory.Type(), factory.DefaultConfig())
	}

	c := v.AllSettings()
	bs, _ := yaml.Marshal(c)
	fmt.Println(string(bs))
	// Output:
	// receivers:
	//   altmockreceiver:
	//     address: altMockReceiverAddress
	//     port: 1616
	//   mockreceiver:
	//     address: mockReceiverAddress
	//     port: 616
}

func checkedBuildReceiverFactory(
	t *testing.T,
	receiverType string,
	newDefaultCfg func() interface{},
	newReceiver func(interface{}, consumer.TraceConsumer, *zap.Logger) (receiver.TraceReceiver, error),
) receiver.TraceReceiverFactory {
	factory, err := NewTraceReceiverFactory(receiverType, newDefaultCfg, newReceiver)
	if err != nil {
		t.Fatalf("failed to build factory for %q: %v", receiverType, err)
	}
	return factory
}

type mockReceiverCfg struct {
	Address string `mapstructure:"address"`
	Port    uint16 `mapstructure:"Port"`
}

type mockReceiver struct {
	config *mockReceiverCfg
}

func newMockReceiverDefaultCfg() interface{} {
	return &mockReceiverCfg{
		Address: "mockReceiverAddress",
		Port:    616,
	}
}

func altNewMockReceiverDefaultCfg() interface{} {
	return &mockReceiverCfg{
		Address: "altMockReceiverAddress",
		Port:    1616,
	}
}

func newMockTraceReceiver(cfg interface{}, next consumer.TraceConsumer, logger *zap.Logger) (receiver.TraceReceiver, error) {
	return &mockReceiver{
		config: cfg.(*mockReceiverCfg),
	}, nil
}

func newMockMetricsReceiver(cfg interface{}, next consumer.MetricsConsumer, logger *zap.Logger) (receiver.MetricsReceiver, error) {
	return &mockReceiver{
		config: cfg.(*mockReceiverCfg),
	}, nil
}

var _ receiver.TraceReceiver = (*mockReceiver)(nil)
var _ receiver.MetricsReceiver = (*mockReceiver)(nil)

func (mr *mockReceiver) TraceSource() string {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return "mockReceiver"
}

func (mr *mockReceiver) StartTraceReception(ctx context.Context, nextConsumer consumer.TraceConsumer) error {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return nil
}

func (mr *mockReceiver) StopTraceReception(ctx context.Context) error {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return nil
}

func (mr *mockReceiver) MetricsSource() string {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return "mockReceiver"
}

func (mr *mockReceiver) StartMetricsReception(ctx context.Context, nextConsumer consumer.MetricsConsumer) error {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return nil
}

func (mr *mockReceiver) StopMetricsReception(ctx context.Context) error {
	if mr == nil {
		panic("mockReceiver is nil")
	}
	return nil
}
