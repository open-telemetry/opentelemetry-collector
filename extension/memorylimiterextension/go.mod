module go.opentelemetry.io/collector/extension/memorylimiterextension

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.49.0
	go.opentelemetry.io/collector/component/componenttest v0.143.0
	go.opentelemetry.io/collector/confmap v1.49.0
	go.opentelemetry.io/collector/extension v1.49.0
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.143.0
	go.opentelemetry.io/collector/extension/extensiontest v0.143.0
	go.opentelemetry.io/collector/internal/memorylimiter v0.143.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.78.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20251013123823-9fd1530e3ec3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/shirou/gopsutil/v4 v4.25.12 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/featuregate v1.49.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.0.0-20260119102301-ba2b601a7080 // indirect
	go.opentelemetry.io/collector/pdata v1.49.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/component => ../../component

replace go.opentelemetry.io/collector/component/componenttest => ../../component/componenttest

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/extension => ../../extension

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/internal/memorylimiter => ../../internal/memorylimiter

replace go.opentelemetry.io/collector/extension/extensiontest => ../../extension/extensiontest

replace go.opentelemetry.io/collector/featuregate => ../../featuregate

replace go.opentelemetry.io/collector/internal/testutil => ../../internal/testutil

replace go.opentelemetry.io/collector/extension/extensionmiddleware => ../../extension/extensionmiddleware

replace go.opentelemetry.io/collector/internal/componentalias => ../../internal/componentalias
