module github.com/open-telemetry/opentelemetry-service

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	contrib.go.opencensus.io/exporter/ocagent v0.5.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	contrib.go.opencensus.io/resource v0.1.1
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.0
	github.com/go-kit/kit v0.8.0
	github.com/gogo/googleapis v1.2.0 // indirect
	github.com/golang/protobuf v1.3.1
	github.com/google/go-cmp v0.3.0
	github.com/gorilla/mux v1.6.2
	github.com/grpc-ecosystem/grpc-gateway v1.9.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jaegertracing/jaeger v1.9.0
	github.com/jstemmer/go-junit-report v0.0.0-20190106144839-af01ea7f8024
	github.com/omnition/scribe-go v0.0.0-20190131012523-9e3c68f31124
	github.com/openzipkin/zipkin-go v0.1.6
	github.com/orijtech/prometheus-go-metrics-exporter v0.0.3-0.20190313163149-b321c5297f60
	github.com/pkg/errors v0.8.0
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/common v0.4.0
	github.com/prometheus/procfs v0.0.0-20190507164030-5867b95ac084
	github.com/prometheus/prometheus v0.0.0-20190131111325-62e591f928dd
	github.com/rs/cors v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.4.0
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-lib v2.0.0+incompatible
	github.com/uber/tchannel-go v1.10.0
	go.opencensus.io v0.22.0
	go.uber.org/zap v1.10.0
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3
	google.golang.org/api v0.5.0
	google.golang.org/grpc v1.21.0
	gopkg.in/yaml.v2 v2.2.2
)
