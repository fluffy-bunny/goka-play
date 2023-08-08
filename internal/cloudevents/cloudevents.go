package cloudevents

import (
	"context"
	"encoding/json"

	cloudevents_pb "github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	xid "github.com/rs/xid"
	zerolog "github.com/rs/zerolog"
)

type (
	SomeCustomData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
)

func MakeRandomCloudEvent(ctx context.Context) (*cloudevents_pb.CloudEvent, error) {
	log := zerolog.Ctx(ctx).With().Logger()
	guid := xid.New()
	customData := &SomeCustomData{
		Name: "John Doe",
		Age:  42,
	}
	cdB, err := json.Marshal(customData)
	if err != nil {
		log.Error().Err(err).Msgf("error marshalling custom data: %v", err)
	}
	return &cloudevents_pb.CloudEvent{
		Id:          guid.String(),
		Source:      "http://example.com",
		Type:        "com.example.test",
		SpecVersion: "1.0",
		Attributes: map[string]*cloudevents_pb.CloudEventAttributeValue{
			"orgid": {
				Attr: &cloudevents_pb.CloudEventAttributeValue_CeString{
					CeString: "ORGTest",
				},
			},
			"canary": {
				Attr: &cloudevents_pb.CloudEventAttributeValue_CeBoolean{
					CeBoolean: true,
				},
			},
			"content-type": {
				Attr: &cloudevents_pb.CloudEventAttributeValue_CeString{
					CeString: "application/json",
				},
			},
		},
		Data: &cloudevents_pb.CloudEvent_TextData{TextData: string(cdB)},
	}, nil
}
