package main

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"
	"syscall"
	"time"

	sarama "github.com/IBM/sarama"
	internal_cloudevents "github.com/fluffy-bunny/goka-play/internal/cloudevents"
	internal_codec "github.com/fluffy-bunny/goka-play/internal/codec"
	internal_logger "github.com/fluffy-bunny/goka-play/internal/logger"
	goka "github.com/lovoo/goka"
	goka_codec "github.com/lovoo/goka/codec"
	zerolog "github.com/rs/zerolog"
)

var (
	brokers  = []string{"herb-event-hub.servicebus.windows.net:9093"}
	password = "Endpoint=sb://herb-event-hub.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=QEHgyc1cK7K4V+nR0RTDQ73+iUVXr4tt7+AEhLQcwQ4="

	topic goka.Stream = "cloudevents"
	group goka.Group  = "cloudevents-group"

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

func getSASLConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Net.SASL.Enable = true
	config.Net.SASL.User = "$ConnectionString"
	config.Net.SASL.Password = password
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
		ClientAuth:         0,
	}
	config.Version = sarama.V1_0_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewReferenceHashPartitioner
	config.Producer.Compression = sarama.CompressionNone

	return config
}

// emits a single message and leave
func runEmitter(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Logger()

	saramaConfig := getSASLConfig()

	emitter, err := goka.NewEmitter(brokers,
		topic,
		new(internal_codec.CloudEvent),
		goka.WithEmitterLogger(internal_logger.NewGoKaZerolog(ctx)),
		goka.WithEmitterProducerBuilder(goka.ProducerBuilderWithConfig(saramaConfig)),
	)
	if err != nil {
		log.Fatal().Err(err).Msgf("error creating emitter: %v", err)
	}
	defer emitter.Finish()
	ce, err := internal_cloudevents.MakeRandomCloudEvent(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msgf("error creating cloud event: %v", err)
	}

	err = emitter.EmitSync("some-key", ce)
	if err != nil {
		log.Fatal().Err(err).Msgf("error emitting message: %v", err)
	}
	log.Print("message emitted")
}

// process messages until ctrl-c is pressed
func runProcessor(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Logger()

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
	saramaConfig := getSASLConfig()
	p, err := goka.NewProcessor(brokers,
		g,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithConfig(saramaConfig, tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(saramaConfig)),
		goka.WithConsumerSaramaBuilder(goka.SaramaConsumerBuilderWithConfig(saramaConfig)),
		goka.WithProducerBuilder(goka.ProducerBuilderWithConfig(saramaConfig)),
		goka.WithLogger(internal_logger.NewGoKaZerolog(ctx)),
	)
	if err != nil {
		log.Fatal().Msgf("error creating processor: %v", err)
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
	ctx := context.Background()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// create a logger and add it to the context
	log := zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
	ctx = log.WithContext(ctx)

	config := goka.DefaultConfig()
	// since the emitter only emits one message, we need to tell the processor
	// to read from the beginning
	// As the processor is slower to start than the emitter, it would not consume the first
	// message otherwise.
	// In production systems however, check whether you really want to read the whole topic on first start, which
	// can be a lot of messages.
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig := getSASLConfig()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	goka.ReplaceGlobalConfig(saramaConfig)

	/*
		// this works
		producer, err := NewProducerForAzureEventHub()
		if err != nil {
			log.Fatal().Msgf("Error creating producer: %v", err)
		}
		part, off, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic:     "cloudevents",
			Key:       sarama.StringEncoder("test"),
			Value:     sarama.StringEncoder("some data"),
			Timestamp: time.Now(),
		})
		if err != nil {
			log.Fatal().Msgf("Error sending message: %v", err)
		}
		log.Printf("Message sent to partition %d at offset %d", part, off)
		return
	*/
	tm, err := goka.NewTopicManager(brokers, saramaConfig, tmc)
	if err != nil {
		log.Fatal().Msgf("Error creating topic manager: %v", err)
	}
	defer tm.Close()
	err = tm.EnsureStreamExists(string(topic), 8)
	if err != nil {
		log.Printf("Error creating kafka topic %s: %v", topic, err)
	}

	runEmitter(ctx)   // emits one message and stops
	runProcessor(ctx) // press ctrl-c to stop
}

// NewProducerForAzureEventHub creates a producer for Azure event hub based on destination config
func NewProducerForAzureEventHub() (sarama.SyncProducer, error) {

	config := getSASLConfig()

	producer, err := sarama.NewSyncProducer(brokers, config)

	return producer, err
}
