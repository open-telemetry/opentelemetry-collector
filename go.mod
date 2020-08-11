module go.opentelemetry.io/collector

go 1.14

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.1-0.20190430175949-e8b55949d948
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/Shopify/sarama v1.26.4
	github.com/apache/thrift v0.13.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/client9/misspell v0.3.4
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/go-kit/kit v0.10.0
	github.com/gogo/googleapis v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.4.2
	github.com/golangci/golangci-lint v1.30.0
	github.com/google/addlicense v0.0.0-20200622132530-df58acafd6d5
	github.com/google/go-cmp v0.5.1
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/jaegertracing/jaeger v1.18.2-0.20200707061226-97d2319ff2be
	github.com/joshdk/go-junit v0.0.0-20200702055522-6efcf4050909
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mjibson/esc v0.2.0
	github.com/open-telemetry/opentelemetry-proto v0.4.0
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/orijtech/prometheus-go-metrics-exporter v0.0.5
	github.com/ory/go-acc v0.2.5
	github.com/pavius/impi v0.0.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.11.1
	github.com/prometheus/procfs v0.1.3
	github.com/prometheus/prometheus v1.8.2-0.20200626085723-c448ada63d83
	github.com/rs/cors v1.7.0
	github.com/securego/gosec v0.0.0-20200316084457-7da9f46445fd
	github.com/shirou/gopsutil v0.0.0-20200517204708-c89193f22d93 // c89193f22d9359848988f32aee972122bb2abdc2
	github.com/soheilhy/cmux v0.1.4
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/tcnksm/ghr v0.13.0
	github.com/tinylib/msgp v1.1.2
	github.com/uber/jaeger-lib v2.2.0+incompatible
	go.opencensus.io v0.22.4
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae
	golang.org/x/text v0.3.3
	google.golang.org/api v0.29.0 // indirect
	google.golang.org/genproto v0.0.0-20200624020401-64a14ca9d1ad
	google.golang.org/grpc v1.31.0
	google.golang.org/grpc/examples v0.0.0-20200728065043-dfc0c05b2da9 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	honnef.co/go/tools v0.0.1-2020.1.5
)
