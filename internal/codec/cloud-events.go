package codec

import (
	"fmt"

	cloudevents_pb "github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

// CloudEvent codec is
type CloudEvent struct{}

// Encode does a type conversion into []byte
func (d *CloudEvent) Encode(value interface{}) ([]byte, error) {
	cloudEvent, isCloudEvent := value.(*cloudevents_pb.CloudEvent)
	var err error

	if !isCloudEvent {
		err = fmt.Errorf("DefaultCodec: value to encode is not of type []byte")
		return nil, err
	}
	data, err := protojson.Marshal(cloudEvent)
	return data, err
}

// Decode of defaultCodec simply returns the data
func (d *CloudEvent) Decode(data []byte) (interface{}, error) {
	cloudEvent := &cloudevents_pb.CloudEvent{}
	err := protojson.Unmarshal(data, cloudEvent)
	if err != nil {
		return nil, err
	}
	return cloudEvent, nil
}
