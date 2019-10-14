# OpenTelemetry UNIX Domain Socket receiver

The UNIX Domain Socket (UDS) receiver allows libraries such as the OpenCensus PHP library
to export stats and traces  over a local unix socket or windows named pipe. The UDS receiver
then takes care  of transporting the information to the OpenTelemetry Collector.

By using the UDS receiver in between the OpenCensus PHP library and OpenTelemetry collector
we avoid the need of injecting the gRPC extension into PHP, even making it 
possible to run OpenTelemetry/OpenCensus without the need of installing any extension.

## Daemon Protocol

Communication between OpenCensus PHP and the UDS receiver is 
handled by using a simple protocol. The protocol is designed to expect partial 
message delivery allowing such a message to be discarded and proceed to ingest 
the next message. This situation can occur when using the PHP user land 
implementation of the Daemon client as it uses non blocking sockets with a 
settable maximum message delivery time to not hold up the PHP request critical 
path.

A protocol message consists of a header and the message payload.

All number representations (UVarint, Float64, padded Float32, Integer) are sent 
in big endian (network) byte order.

The original design is in this [design doc](https://docs.google.com/document/d/1CRiRq_wpzOuG9VKM_eaImcrS12Iie2V7LePnH9AwclU/)

### Protocol header

| name           | type    | size  | comment                                                   
|----------------|---------|------:|---------
| SOM            | Bytes   |     4 | Start of message "\x00\x00\x00\x00"
| Message Type   | Byte    |     1 | Message Type
| Sequence Nr.   | UVarInt |  1-10 |
| Process ID     | UVarInt |  1-10 |
| Thread ID      | UVarInt |  1-10 | Will be 0 for non ZTS deployments
| StartTime      | Float*  |     8 | Unix timestamp with fraction: Float64 or padded Float32 (`\x00\x00 + Float32 + \x00\x00`)
| Message Length | UVarInt |  1-10 | **TODO:** identify a maximum allowed message length. This will most likely be dictated by Span exports.

**Note:** PHP can't guarantee the size of a Float when exporting it out. It will 
either be 64 bit or 32 bit depending on the platform. If StartTime is padded 
with `\x00\x00` on both sides we know that all exported floats will be of 32 bit 
length. If StartTime is a 64bit float, all remaining Float values will be 
exported as 64 bit.

By use of UVarInt we have a variable length Header. This allows the header to be 
as small as 17 bytes in the most optimistic circumstances and as large as 53 in 
the most pessimistic ones.

### Data Types

Our messages allow for the following data types and their encoding:

#### Byte
Single unsigned byte holding value between 0 and 255.

#### Float
Depending on the PHP runtime a float value will be either 64 or 32 bit in 
resolution. All floats are big endian (network) byte order encoded. To identify
the size of the float, the Daemon implementation inspects the StartTime field in 
the header. If this field is `\x00\x00` padded on both sides, we're dealing with 
32 bit floats.

#### String
Strings are prefixed by a UVarInt indicating string length immediately followed 
by the unterminated binary string payload. Empty strings are allowed.

#### Array
Arrays are prefixed by a UVarInt indicating the array item count immediately 
followed by the array items. Empty arrays are allowed.

### Protocol message types

The following message types and their contents have been defined.

#### MSG_PROC_INIT
**Message Type: 1**

Sent to the Daemon when a new PHP Process spawns. (only sent if using the 
OpenCensus/OpenTelemetry Extension).

**TODO**: payload to be determined 

#### MSG_PROC_SHUTDOWN
**Message Type: 2**

Sent to the Daemon when a PHP Process is shut down. (only sent if using the 
OpenCensus/OpenTelemetry Extension).

**TODO**: payload to be determined 

#### MSG_REQ_INIT
**Message Type: 3**

Sent to the Daemon when a new PHP Request is received.

| field              | type    | comment
|--------------------|---------|---------
| Protocol Version   | Byte    | Daemon Protocol Version (currently 1 only)
| PHP Version        | String  |
| Zend Version       | String  |


#### MSG_REQ_SHUTDOWN
**Message Type: 4**

Sent to the Daemon when a PHP Request ends.

| field | type | comment
|-------|------|---------
|     - |    - | no message payload

#### MSG_TRACE_EXPORT 
**Message Type: 20**

A collection of Spans ready to be exported.

**TODO:** Currently uses a JSON encoded array of spans. Span payload is
subject to change.

#### MSG_MEASURE_CREATE
**Message Type: 40**

Measure Create request.

| field              | type    | comment
|--------------------|---------|---------
| MeasureType        | Byte    | 1 = IntMeasure, 2 = FloatMeasure
| Name               | String  |
| Description        | String  |
| Unit               | String  |

#### MSG_VIEW_REPORTING_PERIOD
**Message Type: 41**

Reporting Period adjustment request.

| field              | type    | comment
|--------------------|---------|---------
| Interval           | Float   | Interval in seconds

#### MSG_VIEW_REGISTER
**Message Type: 42**

Register View request.

| field              | type              | comment
|--------------------|-------------------|---------
| Views              | Array of **View** | Array of Views to Register.

View Type:

| field              | type            | comment
|--------------------|-----------------|---------
| View Name          | String          |
| View Description   | String          |
| Tag Keys           | Array of String | Array of TagKey values.
| Measure Name       | String          |
| Aggregation Type   | Byte            | 0 = None, 1 = Count, 2 = Sum, 3 = Distribution, 4 = Last Value
| Bucket Boundaries* | Array of Float  | Only sent for the Distribution aggregation type


#### MSG_VIEW_UNREGISTER
**Message Type: 43**

Unregister View request.

| field              | type            | comment
|--------------------|-----------------|---------
| View Names         | Array of String | Array holding names of Views to Unregister

#### MSG_STATS_RECORD
**Message Type: 44**

Stats Record request.

| field              | type                     | comment
|--------------------|--------------------------|---------
| Measurements       | Array of **Measurement** | Measurements to record
| Tags               | Array of **KeyValue**  | Tag kv pairs
| Attachments        | Array of **KeyValue**  | Exemplar attachment kv pairs

Measurement Type:

| field        | type   | comment
|--------------|--------|---------
| Name         | String |
| Measure Type | Byte   | 1 = IntMeasure, 2 = FloatMeasure
| Value        | Float  | Value is always transported as Float

KeyValue Type:

| field | type   | comment
|-------|--------|---------
| Key   | String |
| Value | String |
