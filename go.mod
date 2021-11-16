module go.opentelemetry.io/collector

go 1.17

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/gogo/protobuf v1.3.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/knadh/koanf v1.3.2
	github.com/magiconair/properties v1.8.5
	github.com/mitchellh/mapstructure v1.4.2
	github.com/mostynb/go-grpc-compression v1.1.14
	github.com/prometheus/common v0.32.1
	github.com/rs/cors v1.8.0
	github.com/shirou/gopsutil/v3 v3.21.10
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector/model v0.39.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.26.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.26.1
	go.opentelemetry.io/contrib/zpages v0.26.1
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/exporters/prometheus v0.25.0
	go.opentelemetry.io/otel/metric v0.25.0
	go.opentelemetry.io/otel/sdk v1.2.0
	go.opentelemetry.io/otel/sdk/export/metric v0.25.0
	go.opentelemetry.io/otel/sdk/metric v0.25.0
	go.opentelemetry.io/otel/trace v1.2.0
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/sys v0.0.0-20211013075003-97ac67df715c
	google.golang.org/genproto v0.0.0-20210604141403-392c879c8b08
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
)

require golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect

replace go.opentelemetry.io/collector/model => ./model

retract v0.32.0 // Contains incomplete metrics transition to proto 0.9.0, random components are not working.
