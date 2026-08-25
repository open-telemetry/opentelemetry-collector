# Testing for the OpenTelemetry Collector Builder

This is a set of end-to-end tests for the builder. As such, it includes only positive tests, based on the manifest files in this directory. Each manifest is expected to be in a working state and should yield an OpenTelemetry Collector instance that is ready within a time interval. "Ready" is defined by calling its healthcheck endpoint.
