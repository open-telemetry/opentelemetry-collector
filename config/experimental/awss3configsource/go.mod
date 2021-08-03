module example.com/awss3configsource

go 1.16

require (
	github.com/aws/aws-sdk-go v1.40.5
	github.com/knadh/koanf v1.2.0
	github.com/signalfx/splunk-otel-collector v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.31.0
	go.uber.org/zap v1.18.1
)

replace github.com/signalfx/splunk-otel-collector => ../configprovider
