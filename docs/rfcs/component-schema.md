# Component Schema

## Overview

OpenTelemetry components expose configuration as a public API.
This public API is currently expressed directly in Golang structs.

This RFC discusses expressing the public API of components as JSON schema files.

## Past approaches

### `configschema`
The [`configschema`](https://pkg.go.dev/github.com/open-telemetry/opentelemetry-collector-contrib/cmd/configschema) go tool was part of opentelemetry-collector-contrib and used to build a specific YAML schema.

It required to import all go modules as part of schemabuilder's dependencies, which created issues with the build of the repository.

See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/30187

### Golang JSON schema reverse-engineering

This approach consists of reverse-engineering the JSON schema from the Golang config structs.

This approach ran into issues with the Golang generation from JSON schema.

See https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/27003 for historical context.

### mdatagen Config struct generation

This is [an in-progress approach](https://github.com/open-telemetry/opentelemetry-collector/pull/13155) where the metadata.yaml file of the component contains a config key, under which a JSON schema is present.

This approach is demonstrated with a POC showing the batchprocessor configuration.

Issues are present when it comes to composing configuration by embedding a library such as confighttp.

### checkapi schema check

This is an [in-progress approach](https://github.com/open-telemetry/opentelemetry-go-build-tools/pull/1148) where checkapi, a tool built as part of opentelemetry-go-build-tools, is used to check that
the schema present under the config key of metadata.yaml matches the Config struct fields.

This approach limits to the presence of fields in the Config struct.

The tool can output the missing elements of the schema ; as such it can give a first take on the schema of the component.

The tool also can associate imports such as confighttp.ClientConfig to a reference to a remote schema.

## Goals

We want to use JSON schema to help formulate and stabilize the public API of the components.

JSON schema consumers use it to:
* Create component configurations.
* Validate component configurations.
* Version the configuration of components.

See https://github.com/open-telemetry/opentelemetry-collector/issues/9769 for discussion of the goals.

See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/42214 for a more recent discussion of goals and roadmap.

## Current behavior

### Validation via the `validate` command

The collector can run with a validate CLI option that runs validation of the yaml file.

### Configuration SIG

There is a [SIG](https://github.com/open-telemetry/opentelemetry-configuration) dedicated to OpenTelemetry Configuration ; members of the SIG should review and sign off on this approach.

## Roadmap

To integrate JSON schema into components, we will need to make incremental changes with an opt-in approach.

### Initial schemas

We work to add initial schemas by hand in libraries in opentelemetry-collector.

The schemas are added to the library metadata.yaml files.

[A PR is open to implement this.](https://github.com/open-telemetry/opentelemetry-collector/pull/13726)

### CheckAPI - optional

We upgrade checkapi to 0.27.1 and turn on schema validation in metadata.yaml if present.

We add configuration in checkapi to add references to schemas defined in the previous step.

### Add a schema to each component

Create a schema for all components opting in, starting with simple components and adding more complex ones that provide validation of the approach.

### Tie schema configuration to stability levels

We add to the stability level requirements of the stable level the need to set a configuration schema for the component.