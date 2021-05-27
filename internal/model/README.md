# Protocols

This package provides common ways for decoding serialized bytes into protocol-specific in-memory data models (e.g. Zipkin Span). These data models can then be translated to internal pdata representations. Similarly, pdata can be translated from a data model which can then be serialized into bytes.

**Encoding**: Common interfaces for serializing/deserializing bytes from/to protocol-specific data models.

**Translation**: Common interfaces for translating protocol-specific data models from/to pdata.

**Marshaling**: Common higher level APIs that do both encoding and translation of bytes and data model if going directly pdata ⇔ bytes.
