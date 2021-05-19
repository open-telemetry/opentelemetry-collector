# Protocols

This package provides common ways for decoding serialized bytes into protocol-specific in-memory data models (e.g. Zipkin Span). These data models can then be decoded into internal pdata representations. Similarly, pdata can be encoded into a data model which can then be encoded into bytes.

[encoding](encoding): Common interfaces for encoding/decoding bytes from/to data models.

[translation](translation): Common interfaces for encoding/decoding data models from/to pdata.

This package provides higher level APIs that do both encoding of bytes and data model if going directly pdata <-> bytes.
