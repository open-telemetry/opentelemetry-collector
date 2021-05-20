# File Storage

> :construction: This extension is in alpha. Configuration and functionality are subject to change.

The File Storage extension can persist state to the local file system. 

The extension requires read and write access to a directory. A default directory can be used, but it must already exist in order for the extension to operate.

`directory` is the relative or absolute path to the dedicated data storage directory. 

`timeout` is the maximum time to wait for a file lock. This value does not need to be modified in most circumstances.


```
extensions:
  file_storage:
  file_storage/all_settings:
    directory: /var/lib/otelcol/mydir
    timeout: 1s

service:
  extensions: [file_storage, file_storage/all_settings]
  pipelines:
    traces:
      receivers: [nop]
      processors: [nop]
      exporters: [nop]

# Data pipeline is required to load the config.
receivers:
  nop:
processors:
  nop:
exporters:
  nop:
```
