# goka-play

[goka](https://github.com/lovoo/goka)

## Examples

The same examples in [goka](https://github.com/lovoo/goka/tree/master/examples) but using [CloudEvents](https://github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb/cloudevent.pb.go) as the message format.  

The [codec](./internal/codec/cloud-events.go) is using ```protojson``` to marshal/unmarshal the CloudEvent message.  
