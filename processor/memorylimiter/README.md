# Memory Limiter Processor

Supported pipeline types: metrics, traces

The memory limiter processor is used to prevent out of memory situations on
the collector. Given that the amount and type of data a collector processes is
environment specific and resource utilization of the collector is also dependent
on the configured processors, it is important to put checks in place regarding
memory usage. The memory_limiter processor offers the follow safeguards:

- Ability to define an interval when memory usage will be checked and if memory
usage exceeds a defined limit will trigger GC to reduce memory consumption.
- Ability to define an interval when memory usage will be compared against the
previous interval's value and if the delta exceeds a defined limit will trigger
GC to reduce memory consumption.

In addition, there is a command line option (`mem-ballast-size-mib`) which can be
used to define a ballast, which allocates memory and provides stability to the
heap. If defined, the ballast increases the base size of the heap so that GC
triggers are delayed and the number of GC cycles over time is reduced. While the
ballast is configured via the command line, today the same value configured on the
command line must also be defined in the memory_limiter processor.

Note that while these configuration options can help mitigate out of memory
situations, they are not a replacement for properly sizing and configuring the
collector. For example, if the limit or spike thresholds are crossed, the collector
will return errors to all receive operations until enough memory is freed. This may
result in dropped data.

It is highly recommended to configure the ballast command line option as well as the
memory_limiter processor on every collector. The ballast should be configured to
be 1/3 to 1/2 of the memory allocated to the collector. The memory_limiter
processor should be the first processor defined in the pipeline (immediately after
the receivers). This is to ensure that backpressure can be sent to applicable
receivers and minimize the likelihood of dropped data when the memory_limiter gets
triggered.

Please refer to [config.go](./config.go) for the config spec.

The following configuration options **must be changed**:
- `check_interval` (default = 0s): Time between measurements of memory
usage. Values below 1 second are not recommended since it can result in
unnecessary CPU consumption.
- `limit_mib` (default = 0): Maximum amount of memory, in MiB, targeted to be
allocated by the process heap. Note that typically the total memory usage of
process will be about 50MiB higher than this value.
- `spike_limit_mib` (default = 0): Maximum spike expected between the
measurements of memory usage. The value must be less than `limit_mib`.
- `limit_percentage` (default = 0): Maximum amount of total memory, in percents, targeted to be
allocated by the process heap. This configuration is supported on Linux systems with cgroups
and it's intended to be used in dynamic platforms like docker.
This option is used to calculate `memory_limit` from the total available memory.
For instance setting of 75% with the total memory of 1GiB will result in the limit of 750 MiB.  
The fixed memory setting (`limit_mib`) takes precedence
over the percentage configuration.
- `spike_limit_percentage` (default = 0): Maximum spike expected between the
measurements of memory usage. The value must be less than `limit_percentage`.
This option is used to calculate `spike_limit_mib` from the total available memory.
For instance setting of 25% with the total memory of 1GiB will result in the spike limit of 250MiB.
This option is intended to be used only with `limit_percentage`.

The following configuration options can also be modified:
- `ballast_size_mib` (default = 0): Must match the `mem-ballast-size-mib`
command line option.

Examples:

```yaml
processors:
  memory_limiter:
    ballast_size_mib: 2000
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
```

```yaml
processors:
  memory_limiter:
    ballast_size_mib: 2000
    check_interval: 5s
    limit_percentage: 50
    spike_limit_percentage: 30
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
