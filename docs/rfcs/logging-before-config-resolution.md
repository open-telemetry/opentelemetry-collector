# How To Log Before Config Resolution

## Overview

The OpenTelemetry Collector supports configuring a primary logger that the collector and its components use to write logs.
This logger cannot be created until the user's configuration has been completely resolved.
There is a need to write logs during the collector start-up, before the primary logger is instantiated, such as during
configuration resolution or config validation. This document describes

- why providing logging capabilities during startup is important
- the current (as of v0.99.0) behavior of the Collector
- different solutions to the problem
- the accepted solution

## Why Logging During Startup is Important

When the collector is starting it tries to resolve user configuration as quickly as possible.
But the Collector's configuration resolution strategy is not trivial - it allows for complex interactions between
multiple, different config sources that must all resolve without error. During this process important information could
be shared with users such as:
- [Warnings about deprecated syntax](https://github.com/open-telemetry/opentelemetry-collector/issues/9162)
- [Warnings about undesired, but handled, situations](https://github.com/open-telemetry/opentelemetry-collector/issues/5615)
- Debug information

## Requirements for any solution

1. Once the primary logger is instantiated, it should be usable anywhere in the Collector that does logging.
2. Log timestamps must be accurate to when the log was written, regardless of when the log is written to the user-specified location(s).
3. The EntryCaller (what line number the log originates from) must be accurate to where the log was written.
4. If an error causes the collector to gracefully terminate before the primary logger is created any previously written logs MUST be written to the either stout or stderr if they have not already been written there.

## Current behavior

As of v0.99.0, the collector does not provide a way to log before the primary logger is instantiated.

## Solutions

### Buffer Logs in Memory and Write Them Once the Primary Logger Exists

The Collector could provide a temporary logger that, when written to, keeps the logs in memory. These logs could then
be passed to the primary logger to be written out in the properly configured format/level.

Benefits:
- Logs are written in the user-specified format/level

Downsides:
- If the primary logger is used to write any logs before the buffered logs are passed, logs may be out of order. There are no guarantees that logs will be written in order, so the log timestamps should be taken as the source of truth for ordering.

### Create a Logger Using the Primary Logger's Defaults

When the user provides no primary logger configuration the Collector creates a Logger using a set of default values.
The Collector could, very early in startup, create a logger using these exact defaults and use it until the primary
logger is instantiated.

Benefits:
- Logs order is preserved
- Logs are still written when an error occurs before the primary logger can be instantiated

Downsides:
- Logs may be written in a format/level that differs from the format/level of the primary logger

## Accepted Solution

[Buffer Logs in Memory and Write Them Once the Primary Logger Exists](#buffer-logs-in-memory-and-write-them-once-the-primary-logger-exists)

This solution, while more complex, allows the collector to write out the logs in the user-specified format whenever possible.  A fallback logger must be used in situations where the primary logger could not be created and the collector is shutting down, such as when encountering an error during configuration resolution, but otherwise the primary logger will be used to write logs that occurred before the primary logger existed.
