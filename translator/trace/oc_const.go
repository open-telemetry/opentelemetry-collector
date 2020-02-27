package tracetranslator

// OTLP attributes to map certain OpenCensus proto fields. These fields don't have
// corresponding fields in OTLP, nor are defined in OTLP semantic conventions.
// TODO: decide if any of these must be in OTLP semantic conventions.
const (
	ocAttributeProcessStartTime  = "opencensus.starttime"
	ocAttributeProcessID         = "opencensus.pid"
	ocAttributeExporterVersion   = "opencensus.exporterversion"
	ocAttributeResourceType      = "opencensus.resourcetype"
	ocTimeEventMessageEventType  = "opencensus.timeevent.messageevent.type"
	ocTimeEventMessageEventID    = "opencensus.timeevent.messageevent.id"
	ocTimeEventMessageEventUSize = "opencensus.timeevent.messageevent.usize"
	ocTimeEventMessageEventCSize = "opencensus.timeevent.messageevent.csize"
)
