# Protocols

This package provides common ways for decoding serialized bytes into protocol-specific in-memory data models (e.g. Zipkin Span). These data models can then be translated to internal pdata representations. Similarly, pdata can be translated from a data model which can then be serialized into bytes.

[serializer](serializer): Common interfaces for serializing/deserializing bytes from/to protocol-specific data models.

[translator](translator): Common interfaces for translating protocol-specific data models from/to pdata.

This package provides higher level APIs that do both encoding of bytes and data model if going directly pdata ⇔ bytes.
