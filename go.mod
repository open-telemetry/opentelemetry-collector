module go.opentelemetry.io/collector

go 1.16

require (
	contrib.go.opencensus.io/exporter/prometheus v0.3.0
	github.com/Shopify/sarama v1.29.1
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/antonmedv/expr v1.8.9
	github.com/apache/thrift v0.14.2
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/structtag v1.2.0
	github.com/go-kit/kit v0.11.0
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/jaegertracing/jaeger v1.24.0
	github.com/knadh/koanf v1.1.1
	github.com/leoluk/perflib_exporter v0.1.0
	github.com/magiconair/properties v1.8.5
	github.com/mitchellh/mapstructure v1.4.1
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.29.0
	github.com/prometheus/prometheus v1.8.2-0.20210621150501-ff58416a0b02
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/rs/cors v1.8.0
	github.com/shirou/gopsutil v3.21.6+incompatible
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tklauser/go-sysconf v0.3.6 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible
	github.com/xdg-go/scram v1.0.2
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector/model v0.30.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.21.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/oteltest v1.0.0-RC1
	go.opentelemetry.io/otel/trace v1.0.0-RC1
	go.uber.org/atomic v1.8.0
	go.uber.org/zap v1.18.1
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20210604141403-392c879c8b08
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
)

replace go.opentelemetry.io/collector/model => ./model
