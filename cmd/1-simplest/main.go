package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	sarama "github.com/IBM/sarama"
	cloudevents_pb "github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	internal_codec "github.com/fluffy-bunny/goka-play/internal/codec"
	goka "github.com/lovoo/goka"
	goka_codec "github.com/lovoo/goka/codec"
)

var (
	brokers             = []string{"localhost:9093"}
	topic   goka.Stream = "example-stream"
	group   goka.Group  = "example-group"

	tmc *goka.TopicManagerConfig
)

type (
	SomeCustomData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
)

func init() {
	// This sets the default replication to 1. If you have more then one broker
	// the default configuration can be used.
	tmc = goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	tmc.Stream.Replication = 1
}

// emits a single message and leave
func runEmitter() {
	emitter, err := goka.NewEmitter(brokers, topic, new(internal_codec.CloudEvent))
	if err != nil {
		log.Fatalf("error creating emitter: %v", err)
	}
	defer emitter.Finish()
	customData := &SomeCustomData{
		Name: "John Doe",
		Age:  42,
	}
	cdB, err := json.Marshal(customData)
	if err != nil {
		log.Fatalf("error marshalling custom data: %v", err)
	}
	ce := &cloudevents_pb.CloudEvent{
		Id:          "1234",
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
	}
	err = emitter.EmitSync("some-key", ce)
	if err != nil {
		log.Fatalf("error emitting message: %v", err)
	}
	log.Println("message emitted")
}

// process messages until ctrl-c is pressed
func runProcessor() {
	// process callback is invoked for each message delivered from
	// "example-stream" topic.
	cb := func(ctx goka.Context, msg interface{}) {
		var counter int64
		// ctx.Value() gets from the group table the value that is stored for
		// the message's key.
		if val := ctx.Value(); val != nil {
			counter = val.(int64)
		}
		counter++
		// SetValue stores the incremented counter in the group table for in
		// the message's key.
		ctx.SetValue(counter)
		log.Printf("key = %s, counter = %v, msg = %v", ctx.Key(), counter, msg)
	}

	// Define a new processor group. The group defines all inputs, outputs, and
	// serialization formats. The group-table topic is "example-group-table".
	g := goka.DefineGroup(group,
		goka.Input(topic, new(internal_codec.CloudEvent), cb),
		goka.Persist(new(goka_codec.Int64)),
	)

	p, err := goka.NewProcessor(brokers,
		g,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.DefaultConsumerGroupBuilder),
	)
	if err != nil {
		log.Fatalf("error creating processor: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err = p.Run(ctx); err != nil {
			log.Printf("error running processor: %v", err)
		}
	}()

	sigs := make(chan os.Signal)
	go func() {
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	}()

	select {
	case <-sigs:
	case <-done:
	}
	cancel()
	<-done
}

func main() {
	config := goka.DefaultConfig()
	// since the emitter only emits one message, we need to tell the processor
	// to read from the beginning
	// As the processor is slower to start than the emitter, it would not consume the first
	// message otherwise.
	// In production systems however, check whether you really want to read the whole topic on first start, which
	// can be a lot of messages.
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	goka.ReplaceGlobalConfig(config)

	tm, err := goka.NewTopicManager(brokers, goka.DefaultConfig(), tmc)
	if err != nil {
		log.Fatalf("Error creating topic manager: %v", err)
	}
	defer tm.Close()
	err = tm.EnsureStreamExists(string(topic), 8)
	if err != nil {
		log.Printf("Error creating kafka topic %s: %v", topic, err)
	}

	runEmitter()   // emits one message and stops
	runProcessor() // press ctrl-c to stop
}
