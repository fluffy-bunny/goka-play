# goka-play

[goka](https://github.com/lovoo/goka)

## Examples

The same examples in [goka](https://github.com/lovoo/goka/tree/master/examples) but using  [cloudevents](https://github.com/cloudevents/sdk-go/tree/main/binding/format/protobuf/v2/pb)  as the message format.  

The [codec](./internal/codec/cloud-events.go) is using ```protojson``` to marshal/unmarshal the CloudEvent message.  

## Logger

For these example I am using [zerolog](github.com/rs/zerolog) as the replacement logger for [goka Logger](./internal/logger/goka-zerolog-logger.go).  
