package main

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

type Header struct {
	Action       string `mapstructure:"action"`
	Key          string `mapstructure:"key"`
	FromContext  string `mapstructure:"from_context"`
	DefaultValue string `mapstructure:"default_value"`
}

type HeadersSetterConfig struct {
	Headers []Header `mapstructure:"headers"`
}

type ExtensionsConfig struct {
	HeadersSetter HeadersSetterConfig `mapstructure:"headers_setter"`
}

type Config struct {
	Extensions map[string]interface{} `mapstructure:"extensions"`
}

func main() {
	// Set the environment variable with a numeric value
	os.Setenv("SPLUNK_ACCESS_TOKEN", "12345")

	// Load the config
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{"config.yaml"},
		ProviderFactories: []confmap.ProviderFactory{
			fileprovider.NewFactory(),
		},
	})
	if err != nil {
		fmt.Printf("Error creating resolver: %v\n", err)
		return
	}

	conf, err := resolver.Resolve(context.Background())
	if err != nil {
		fmt.Printf("Error resolving config: %v\n", err)
		return
	}

	// Unmarshal into the config struct
	var config Config
	err = conf.Unmarshal(&config)
	if err != nil {
		fmt.Printf("Error unmarshalling config: %v\n", err)
		return
	}

	fmt.Println("Successfully unmarshalled config!")

	// Extract and verify the header config
	headersSetterConfig, ok := config.Extensions["headers_setter"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: couldn't extract headers_setter config")
		return
	}

	// Convert to confmap to unmarshal
	headerConf := confmap.NewFromStringMap(headersSetterConfig)
	var headerConfig HeadersSetterConfig
	err = headerConf.Unmarshal(&headerConfig)
	if err != nil {
		fmt.Printf("Error unmarshalling headers config: %v\n", err)
		return
	}

	// Verify the DefaultValue is correctly unmarshalled as a string
	if len(headerConfig.Headers) > 0 {
		fmt.Printf("Success! DefaultValue type: %T, value: %s\n",
			headerConfig.Headers[0].DefaultValue,
			headerConfig.Headers[0].DefaultValue)
	} else {
		fmt.Println("Error: No headers found in the configuration")
	}
}
