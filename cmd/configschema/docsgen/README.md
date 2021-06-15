# Docsgen CLI Tool

This package contains a CLI tool that generates markdown files for collector
components. The markdown files present the configuration metadata extracted
by the configschema API in a human readable form that can be used to manually
configure the collector.

## Usage

There are two modes of operation, one where markdown files are created for all
components, and another where a markdown file is created for only one, specified
component.

#### All components

```
docsgen all
```

Creates config.md files in every directory corresponding to a component
configuration type.

#### Single component

```
docsgen component-type component-name
```

Creates a single config.md files in the directory corresponding to the
specified component.

### Usage Example

To create a config doc for the otlp receiver, use the command

```
docsgen receiver otlp
```

This creates a file called `config.md` in `receiver/otlpreceiver`.

### Output Example

[OTLP Receiver Config Metadata Doc](../../../receiver/otlpreceiver/config.md)
