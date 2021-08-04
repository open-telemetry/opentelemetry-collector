# Config Source Manager
The `configsource.Manager` handles the use of multiple [config source objects](../../experimental/configsource/component.go)
simplifying configuration parsing,
monitoring for updates, and the overall life-cycle of the used config sources.

## Usage Overview
The public API of the `configsource.Manager` is described in `godoc` comments at the
[`manager.go`](./manager.go) file.
Below is an overview of its usage, which consists of two separate phases:

1. Configuration Processing
2. Watching for Updates

## Configuration Processing
The `configsource.Manager` receives as input a set of config source factories and a `configparser.Parser` that
will be used to generate the resulting, or effective, configuration also in the form of a `configparser.Parser`,
that can be used by code that is oblivious to the usage of config sources.

```terminal
+-----------------------------------------------------+                                                       
|              configparser.Parser                    |                                                       
|-----------------------------------------------------|                                                       
|                                                     |                                                       
| logical YAML config:                                |                                                       
| +-------------------------------------------------+ |                                                       
| |config_sources:                                  | |                                                       
| |  include:                                       | |                    +---------------------------------+
| |    # `include` is an example of a config        | |                   +---------------------------------+|
| |    # source that can read from files.           | |                  +---------------------------------+||
| |  env:                                           | |                 +---------------------------------+|||
| |    # `env` is another possible config           | |                +---------------------------------+||||
| |    # source that insert YAML from env vars.     | |                |                                 |||||
| |                                                 | |                |                                 |||||
| |# Below the standard YAML configuration but      | |                |                                 |||||
| |# some data still to be retrieved from the       | |                |                                 |||||
| |# config sources.                                | |                |                                 |||||
| |                                                 | |                |      configsource Factory       |||||
| |receivers: ${include:/cfgs/rcvrs/def.yaml}       | |                |                                 |||||
| |                                                 | |                |                                 |||||
| |exporters ${env:EXPORTERS_DEFINITION}            | |                |                                 |||||
| |                                                 | |                |                                 |||||
| |service:                                         | |                |                                 |||||
| |  pipelines:                                     | |                |                                 |||||
| |    trace:                                       | |                |                                 ||||+
| |      receivers: ${include:/cfgs/rcvrs/use.yaml} | |                |                                 |||+ 
| |      exporters: ${env:EXPORTERS_IN_USE}         | |                |                                 ||+  
| +-------------------------------------------------+ |                |                                 |+   
+-----------------------------------------------------+                +---------------------------------+    
                       |                                                                |                     
                       |                                                                |                     
                       +----------------------------------------------------------------+                     
                                                     |                                                        
                                                     |                                    +------------------+
                                                     v                                   +------------------+|
                                     +-------------------------------+                  +------------------+||
                                     |                               |                  |                  |||
                                     |      configsource.Manager     |                  | configsource Obj ||+
                                     |                               |                  |                  |+ 
                                     +-------------------------------+                  +------------------+  
                                                     |       |                                    |           
                                                     |       |                                    |           
                                           "Resolve" |       +------------------------------------+           
                                                     |          "from `config_sources:` section"              
                                                     |                                                        
                                                     |                                                        
                                                     |                                                        
                                                     |                                                        
                                                     v                                                        
                          +-----------------------------------------------------+                             
                          |                 configparser.Parser                 |                             
                          |-----------------------------------------------------|                             
                          |                                                     |                             
                          | logica  YAML config:                                |                             
                          | +-------------------------------------------------+ |                             
                          | |receivers:                                       | |                             
                          | |  zipkin:                                        | |                             
                          | |  jaeger:                                        | |                             
                          | |  otlp:                                          | |                             
                          | |exporters:                                       | |                             
                          | |  zipkin:                                        | |                             
                          | |  jaeger:                                        | |                             
                          | |  otlp:                                          | |                             
                          | |service:                                         | |                             
                          | |  pipelines:                                     | |                             
                          | |    trace:                                       | |                             
                          | |      receivers: [zipkin, jaeger, otlp]          | |                             
                          | |      exporters: [otlp]                          | |                             
                          | +-------------------------------------------------+ |                             
                          +-----------------------------------------------------+                             
                                                                                                              
                                                                                                                                                                                                                            
```

## Watching for Updates
After the configuration was processed the `configsource.Manager` can be used as a single point to watch for updates in
the configuration data retrieved via the config sources used to process the “initial” configuration and to generate
the“effective” one.

The `configsource.Manager` does that by wrapping calls to the `WatchForUpdate()` method of each `configsource.Retreived`
object that was used during the configuration processing. It also controls the lifecycle of all `configsource.Session`
objects created to get the `configsource.Retrieved` objects. 
