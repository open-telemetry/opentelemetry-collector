module go.opentelemetry.io/collector/service

go 1.20

require (
	github.com/google/uuid v1.5.0
	github.com/prometheus/client_golang v1.18.0
	github.com/prometheus/client_model v0.5.0
	github.com/prometheus/common v0.46.0
	github.com/shirou/gopsutil/v3 v3.23.12
	github.com/stretchr/testify v1.8.4
	go.opencensus.io v0.24.0
	go.opentelemetry.io/collector v0.92.0
	go.opentelemetry.io/collector/component v0.92.0
	go.opentelemetry.io/collector/config/confignet v0.92.0
	go.opentelemetry.io/collector/config/configtelemetry v0.92.0
	go.opentelemetry.io/collector/confmap v0.92.0
	go.opentelemetry.io/collector/connector v0.92.0
	go.opentelemetry.io/collector/consumer v0.92.0
	go.opentelemetry.io/collector/exporter v0.92.0
	go.opentelemetry.io/collector/extension v0.92.0
	go.opentelemetry.io/collector/extension/zpagesextension v0.92.0
	go.opentelemetry.io/collector/featuregate v1.0.1
	go.opentelemetry.io/collector/pdata v1.0.1
	go.opentelemetry.io/collector/processor v0.92.0
	go.opentelemetry.io/collector/receiver v0.92.0
	go.opentelemetry.io/collector/semconv v0.92.0
	go.opentelemetry.io/contrib/config v0.2.0
	go.opentelemetry.io/contrib/propagators/b3 v1.22.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/bridge/opencensus v0.45.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.45.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.45.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.22.0
	go.opentelemetry.io/otel/exporters/prometheus v0.45.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.45.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.22.0
	go.opentelemetry.io/otel/metric v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/sdk/metric v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.26.0
	gonum.org/v1/gonum v0.14.0
)

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.0.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	go.opentelemetry.io/contrib/zpages v0.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.22.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/grpc v1.60.1 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector => ../

replace go.opentelemetry.io/collector/connector => ../connector

replace go.opentelemetry.io/collector/component => ../component

replace go.opentelemetry.io/collector/pdata => ../pdata

replace go.opentelemetry.io/collector/extension/zpagesextension => ../extension/zpagesextension

replace go.opentelemetry.io/collector/extension => ../extension

replace go.opentelemetry.io/collector/exporter => ../exporter

replace go.opentelemetry.io/collector/confmap => ../confmap

replace go.opentelemetry.io/collector/config/configtelemetry => ../config/configtelemetry

replace go.opentelemetry.io/collector/processor => ../processor

replace go.opentelemetry.io/collector/consumer => ../consumer

replace go.opentelemetry.io/collector/semconv => ../semconv

replace go.opentelemetry.io/collector/receiver => ../receiver

replace go.opentelemetry.io/collector/featuregate => ../featuregate

replace go.opentelemetry.io/collector/config/confignet => ../config/confignet

replace go.opentelemetry.io/collector/config/configretry => ../config/configretry
