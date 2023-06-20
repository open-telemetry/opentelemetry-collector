module go.opentelemetry.io/collector/config/configtls

go 1.19

require (
	github.com/fsnotify/fsnotify v1.6.0
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/collector/config/configopaque v0.80.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/config/configopaque => ../configopaque
