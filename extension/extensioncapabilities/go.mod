module go.opentelemetry.io/collector/extension/extensioncapabilities

go 1.22.0

require (
	go.opentelemetry.io/collector/component v0.114.0
	go.opentelemetry.io/collector/confmap v1.20.0
	go.opentelemetry.io/collector/extension v0.114.0
)

require (
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.114.0 // indirect
	go.opentelemetry.io/collector/pdata v1.20.0 // indirect
	go.opentelemetry.io/otel v1.32.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/extension => ../

replace go.opentelemetry.io/collector/pdata => ../../pdata

replace go.opentelemetry.io/collector/confmap => ../../confmap

replace go.opentelemetry.io/collector/config/configtelemetry => ../../config/configtelemetry

replace go.opentelemetry.io/collector/component => ../../component
