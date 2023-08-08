package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	internal_cloudevents "github.com/fluffy-bunny/goka-play/internal/cloudevents"
	internal_codec "github.com/fluffy-bunny/goka-play/internal/codec"
	internal_logger "github.com/fluffy-bunny/goka-play/internal/logger"
	"github.com/gogo/status"
	"github.com/hashicorp/go-multierror"
	goka "github.com/lovoo/goka"
	multierr "github.com/lovoo/goka/multierr"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
)

var (
	brokers                  = []string{"localhost:9093"}
	inputTopic   goka.Stream = "input-topic"
	forwardTopic goka.Stream = "forward-topic"

	nocommit = flag.Bool("no-commit", false, "set this to true for testing what happens if we don't commit")

	tmc *goka.TopicManagerConfig
)

func init() {
	// This sets the default replication to 1. If you have more then one broker
	// the default configuration can be used.
	tmc = goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	tmc.Stream.Replication = 1
}

func main() {
	ctx := context.Background()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// create a logger and add it to the context
	log := zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
	ctx = log.WithContext(ctx)

	flag.Parse()

	createTopics(ctx)

	// some channel to stop on signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal)
	go func() {
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		<-sigs
		cancel()
	}()

	errg, ctx := multierr.NewErrGroup(ctx)

	// (1) start the input processor
	errg.Go(func() error {

		inputEmitter, err := goka.NewEmitter(brokers,
			"input-topic", new(internal_codec.CloudEvent),
			goka.WithEmitterLogger(internal_logger.NewGoKaZerolog(ctx)),
		)
		if err != nil {
			log.Fatal().Msgf("error external emitter: %v", err)
		}

		// error dropped here for simplicity
		defer inputEmitter.Finish()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				ce, _ := internal_cloudevents.MakeRandomCloudEvent(ctx)
				// error dropped here for simplicity
				inputEmitter.Emit("time", ce)
			}
		}
	})

	// (2) start the forwarding processor
	errg.Go(func() error {

		asyncAction := func(err error) async.Future[error] {
			promise := async.NewPromise[error]()
			go func() {
				time.Sleep(time.Second * 3)
				if err == nil {
					// yea yea, for clarity
					promise.Success(nil)
				} else {
					promise.Failure(err)
				}

			}()

			return promise.Future()
		}
		// This emitter represents the component that depends on "external" ressources, like a different kafka-cluster,
		// or an emitter to a different message queue or database with async writes etc...
		forwardEmitter, err := goka.NewEmitter(brokers,
			forwardTopic,
			new(internal_codec.CloudEvent),
			goka.WithEmitterLogger(internal_logger.NewGoKaZerolog(ctx)),
		)
		if err != nil {
			log.Fatal().Msgf("error external emitter: %v", err)
		}

		p, err := goka.NewProcessor(brokers,
			goka.DefineGroup("forwarder",
				goka.Input(inputTopic, new(internal_codec.CloudEvent), func(ctx goka.Context, msg interface{}) {

					// forward the incoming message to the "external" emitter
					prom, err := forwardEmitter.Emit("time", msg)
					if err != nil {
						ctx.Fail(fmt.Errorf("error emitting: %v", err))
					}

					commit := ctx.DeferCommit()
					var future async.Future[error]
					if *nocommit {
						err := status.Error(codes.Internal, "something went wrong")
						future = asyncAction(err)
					} else {
						future = asyncAction(nil)
					}
					_, err = future.Join()
					if err == nil {
						prom.Then(commit)
					} else {
						log.Error().Err(err).Msgf("error in async action")
					}

				}),
			),
			goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
			goka.WithLogger(internal_logger.NewGoKaZerolog(ctx)),
		)
		if err != nil {
			log.Fatal().Msgf("error creating processor: %v", err)
		}

		return p.Run(ctx)
	})

	// (3) start the consuming processor
	errg.Go(func() error {

		// processor that simply prints the incoming message.
		p, err := goka.NewProcessor(brokers,
			goka.DefineGroup("consumer",
				goka.Input(forwardTopic, new(internal_codec.CloudEvent), func(ctx goka.Context, msg interface{}) {
					log.Printf("received message %s: %v", ctx.Key(), msg)
				}),
			),
			goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
			goka.WithLogger(internal_logger.NewGoKaZerolog(ctx)),
		)
		if err != nil {
			log.Fatal().Msgf("error creating processor: %v", err)
		}

		return p.Run(ctx)
	})

	if err := errg.Wait().ErrorOrNil(); err != nil {
		log.Fatal().Msgf("Error executing: %v", err)
	}
}

func createTopics(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Logger()

	// create a new topic manager so we can create the streams we need
	tmg, err := goka.NewTopicManager(brokers, goka.DefaultConfig(), tmc)
	if err != nil {
		log.Fatal().Msgf("Error creating topic manager: %v", err)
	}

	// error ignored for simplicity
	defer tmg.Close()
	errs := multierror.Append(
		tmg.EnsureStreamExists(string(inputTopic), 6),
		tmg.EnsureStreamExists(string(forwardTopic), 6),
	)
	if errs.ErrorOrNil() != nil {
		log.Fatal().Msgf("cannot create topics: %v", errs.ErrorOrNil())
	}
}
