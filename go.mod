module github.com/open-telemetry/opentelemetry-collector

go 1.13

require (
	cloud.google.com/go v0.54.0 // indirect
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	contrib.go.opencensus.io/exporter/ocagent v0.6.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/resource v0.1.2
	github.com/apache/thrift v0.13.0
	github.com/armon/go-metrics v0.3.3 // indirect
	github.com/aws/aws-sdk-go v1.29.20 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/client9/misspell v0.3.4
	github.com/go-kit/kit v0.10.0
	github.com/gogo/googleapis v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.3.4
	github.com/golangci/golangci-lint v1.21.0
	github.com/google/addlicense v0.0.0-20190510175307-22550fa7c1b0
	github.com/google/go-cmp v0.4.0
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/gophercloud/gophercloud v0.8.0 // indirect
	github.com/gorilla/handlers v1.4.2 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/grpc-gateway v1.14.2
	github.com/hashicorp/consul/api v1.4.0 // indirect
	github.com/hashicorp/go-hclog v0.12.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.9.0 // indirect
	github.com/jaegertracing/jaeger v1.17.0
	github.com/jmespath/go-jmespath v0.0.0-20200310153549-2d053f87d1d7 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/miekg/dns v1.1.27 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/open-telemetry/opentelemetry-proto v0.0.0-20200308012146-674ae1c8703f
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/orijtech/prometheus-go-metrics-exporter v0.0.3-0.20190313163149-b321c5297f60
	github.com/pavius/impi v0.0.0-20180302134524-c1cbdcb8df2b
	github.com/pkg/errors v0.9.1
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/procfs v0.0.10
	github.com/prometheus/prometheus v1.8.2-0.20190924101040-52e0504f83ea
	github.com/rs/cors v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.1-0.20190911140308-99520c81d86e
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.16.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/uber/tchannel-go v1.10.0
	go.opencensus.io v0.22.3
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073 // indirect
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527
	google.golang.org/genproto v0.0.0-20200310143817-43be25429f5a // indirect
	google.golang.org/grpc v1.27.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/client-go v0.15.10 // indirect
	k8s.io/utils v0.0.0-20200229041039-0a110f9eb7ab // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
