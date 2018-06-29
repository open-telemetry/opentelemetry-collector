[![Build Status](https://travis-ci.org/DataDog/datadog-go.svg?branch=master)](https://travis-ci.org/DataDog/datadog-go)
# Overview

Packages in `datadog-go` provide Go clients for various APIs at [DataDog](http://datadoghq.com).

## Statsd

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/DataDog/datadog-go/statsd)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](http://opensource.org/licenses/MIT)

The [statsd](https://github.com/DataDog/datadog-go/tree/master/statsd) package provides a client for
[dogstatsd](http://docs.datadoghq.com/guides/dogstatsd/):

```go
import "github.com/DataDog/datadog-go/statsd"

func main() {
    c, err := statsd.New("127.0.0.1:8125")
    if err != nil {
        log.Fatal(err)
    }
    // prefix every metric with the app name
    c.Namespace = "flubber."
    // send the EC2 availability zone as a tag with every metric
    c.Tags = append(c.Tags, "region:us-east-1a")
    err = c.Gauge("request.duration", 1.2, nil, 1)
    // ...
}
```

## License

All code distributed under the [MIT License](http://opensource.org/licenses/MIT) unless otherwise specified.
