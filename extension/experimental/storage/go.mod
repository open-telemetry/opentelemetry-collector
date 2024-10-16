module go.opentelemetry.io/collector/extension/experimental/storage

go 1.22.0

require (
	go.opentelemetry.io/collector/component v0.111.0
	go.opentelemetry.io/collector/extension v0.111.0
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.111.0 // indirect
	go.opentelemetry.io/collector/pdata v1.17.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)

replace go.opentelemetry.io/collector/extension => ../../

replace go.opentelemetry.io/collector/component => ../../../component

replace go.opentelemetry.io/collector/pdata => ../../../pdata

replace go.opentelemetry.io/collector/config/configtelemetry => ../../../config/configtelemetry
