# Subprocess manager

**Note:** This subprocess manager was created/tested with a Linux environment in mind. Please proceed with caution when developing with this package for any other OS.

## Features

This subprocess manager can run commands, format environment variables for use by Go's `exec` package, pipe subprocess `stdout` & `stderr` back to `Uber-go Zap` and compute a restart delay using exponential backoff.

## Usage

This package provides one config and two exported functions that can be used to manage a subprocess. 

### Config

`config.go` defines a struct/configuration that is used to keep track of a subprocess' data and that can be embedded directly into another receiver/extension/processor/exporter's config.

### Run

`Run(logger *zap.Logger) (time.Duration, error)` first parses the `Command` string in the struct to separate the executable from the arguments. It then formats the slice of environment variables (`Env`) and prepares the pipes for logging the output of the subprocess. Finally, it executes the command while keeping track of runtime and returns said runtime (and an error if the process exited with one).

### GetDelay

`GetDelay(elapsed time.Duration, healthyProcessDuration time.Duration, crashCount int, healthyCrashCount int) time.Duration` computes exponential delay for a given set of arguments. A small base delay of `1s` is returned if a process is deemed healthy (the runtime `elapsed` is longer than the `healthyProcessDuration` **or** if the process' `crashCount` is smaller or equal than `healthyCrashCount`). If not, the correct delay is computed.

The parent package should keep track of the process' crash count and define values for `healthyProcessDuration` and `healthyCrashCount` to leverage this package fully.
