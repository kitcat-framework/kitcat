package kiteventpg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitevent"
	"github.com/kitcat-framework/kitcat/kitslog"
	"github.com/kitcat-framework/kitcat/pkg/kitpg/pgutils"
	_ "github.com/lib/pq"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"
)

type PostgresEventStoreConfig struct {
	PollInterval time.Duration `cfg:"poll_interval"`
	CreateSchema bool          `cfg:"create_schema"`
}

func (c *PostgresEventStoreConfig) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitevent.config_stores.postgres"

	viper.SetDefault(prefix+".poll_interval", time.Millisecond*500)
	viper.SetDefault(prefix+".create_schema", true)

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal postgres event store config: %w")
}

func init() {
	kitcat.RegisterConfig(new(PostgresEventStoreConfig))
}

type PostgresEventStore struct {
	handlers   map[kitevent.EventName][]kitevent.Consumer
	db         *gorm.DB
	store      EventStoreStorage
	logger     *slog.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc

	config *PostgresEventStoreConfig
}

func New(db *gorm.DB, logger *slog.Logger, config *PostgresEventStoreConfig) *PostgresEventStore {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &PostgresEventStore{
		db: db,
		logger: logger.With(
			kitslog.Module("kitevent"),
			slog.String("store", "postgres")),
		handlers:   make(map[kitevent.EventName][]kitevent.Consumer),
		store:      NewPgEventStore(db),
		ctx:        ctx,
		cancelFunc: cancelFunc,
		config:     config,
	}
}

func (p PostgresEventStore) OnStart(ctx context.Context) error {
	if p.config.CreateSchema {
		err := p.db.Exec("CREATE SCHEMA IF NOT EXISTS kitevent;").Error
		if err != nil {
			return fmt.Errorf("failed to create kitevent schema: %w", err)
		}
	}

	db, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get db instance: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		SchemaName: "kitevent",
	})

	_, currentFile, _, _ := runtime.Caller(0)
	libraryPath := filepath.Dir(currentFile)
	migrationsPath := filepath.Join(libraryPath, "migrations")

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	go p.run(p.ctx)
	go p.monitorTimeoutEvents(p.ctx)

	return nil
}

func (p PostgresEventStore) Produce(ctx context.Context, event kitevent.Event, opt *kitevent.ProducerOptions) error {
	handlersConcerned := p.handlers[event.EventName()]
	if len(handlersConcerned) == 0 {
		return errors.New("no consumer found for event")
	}

	marshalPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshalPayload event: %w", err)
	}

	if opt == nil {
		opt = kitevent.NewProducerOptions()
	}

	evt := Event{
		Payload:   datatypes.JSON(marshalPayload),
		EventName: event.EventName().Name,
		CreatedAt: pgutils.TimestampUTC(time.Now()),
		UpdatedAt: pgutils.TimestampUTC(time.Now()),
	}

	processors := make([]*EventProcessingState, len(handlersConcerned))
	for i, consumer := range handlersConcerned {
		maxRetries := int32(1)
		if consumer.Options().MaxRetries != nil {
			maxRetries = *consumer.Options().MaxRetries
		}

		processableAt := time.Now()
		if opt.ProduceAt != nil {
			processableAt = *opt.ProduceAt
		}

		retryIntervalMs := int64(0)
		if consumer.Options().RetryInterval != nil {
			retryIntervalMs = consumer.Options().RetryInterval.Milliseconds()
		}

		processors[i] = &EventProcessingState{
			ConsumerName:                  consumer.Name(),
			Status:                        EventProcessingStateStatusAvailable,
			Error:                         nil,
			RetryNumber:                   1,
			ConsumerOptionMaxRetries:      maxRetries,
			ConsumerOptionRetryIntervalMs: retryIntervalMs,
			ConsumerOptionTimeoutMs:       consumer.Options().Timeout.Milliseconds(),
			CreatedAt:                     pgutils.TimestampUTC(time.Now()),
			UpdatedAt:                     pgutils.TimestampUTC(time.Now()),
			ProcessableAt:                 pgutils.TimestampUTC(processableAt),
			RunAt:                         nil,
			DurationMs:                    0,
			FailedAt:                      nil,
			SuccessAt:                     nil,
			PendingAt:                     nil,
		}
	}

	err = p.store.AddEvent(ctx, evt, processors)
	if err != nil {
		return fmt.Errorf("failed to add event: %w", err)
	}

	return nil
}

func (p PostgresEventStore) ProduceSync(
	ctx context.Context,
	event kitevent.Event,
	opts *kitevent.ProducerOptions,
) error {
	if opts == nil {
		opts = kitevent.NewProducerOptions()
	}

	handlers, ok := p.handlers[event.EventName()]

	if !ok {
		return nil
	}

	for _, handler := range handlers {
		return kitevent.LocalCallHandler(kitevent.LocalCallConsumerParams{
			Ctx:           ctx,
			Event:         event,
			Producer:      p,
			Opts:          opts,
			Consumer:      handler,
			Logger:        p.logger,
			IsProduceSync: true,
		})
	}

	return nil
}

func (p PostgresEventStore) AddConsumer(eventName kitevent.EventName, handler kitevent.Consumer) {
	p.handlers[eventName] = append(p.handlers[eventName], handler)
}

func (p PostgresEventStore) OnStop(_ context.Context) error {
	p.cancelFunc()

	return nil
}

func (p PostgresEventStore) Name() string {
	return "postgres"
}

// run is a blocking function that will loop over handler results to find those who are in EventProcessingStateStatusAvailable status.
// If one is found, it will call the consumer associated to the event and update the handler result accordingly.
// If none is found, it will wait for 500ms and try again. Basic polling.
func (p PostgresEventStore) run(ctx context.Context) {
	for {
		eventHandlerResult, err := p.nextEvent(ctx)
		if err != nil || eventHandlerResult == nil {
			time.Sleep(p.config.PollInterval)
			continue
		}

		handlers := p.handlers[kitevent.NewEventName(eventHandlerResult.Event.EventName)]
		if len(handlers) == 0 {
			continue
		}

		for _, handler := range handlers {
			if eventHandlerResult.ConsumerName != handler.Name() {
				continue
			}

			event, err := kitevent.PayloadToEvent(handler, eventHandlerResult.Event.Payload)
			if err != nil {
				p.logger.Error("failed to convert payload to event", kitslog.Err(err))
				continue
			}

			// TODO: process them in a pool to avoid having blocking handler
			p.processConsumer(
				ctx,
				event,
				handler,
				eventHandlerResult,
			)

			break
		}
	}
}

func (p PostgresEventStore) monitorTimeoutEvents(ctx context.Context) {
	for {
		evtProcessingState, err := p.nextEventInTimeout(ctx)
		if err != nil || evtProcessingState == nil {
			time.Sleep(p.config.PollInterval)
			continue
		}

		l := p.logger.With(
			slog.Int("event_id", int(evtProcessingState.EventID)),
			slog.String("consumer", evtProcessingState.ConsumerName),
			slog.String("event_name", evtProcessingState.Event.EventName))

		nextEvtProcessingStateResult := []*EventProcessingState{}

		evtProcessingState.UpdatedAt = pgutils.TimestampUTC(time.Now())
		evtProcessingState.RunAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))
		evtProcessingState.DurationMs = time.Since(evtProcessingState.PendingAt.Time).Milliseconds()
		evtProcessingState.Error = lo.ToPtr("consumer timeout reached")
		evtProcessingState.Status = EventProcessingStateStatusFailed
		evtProcessingState.FailedAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))

		if evtProcessingState.RetryNumber >= evtProcessingState.ConsumerOptionMaxRetries {
			l.Error("timout reached, max retries reached",
				slog.Int("retry_number", int(evtProcessingState.RetryNumber)),
				slog.Int("max_retries", int(evtProcessingState.ConsumerOptionMaxRetries)),
				slog.Int64("retry_interval_ms", evtProcessingState.ConsumerOptionRetryIntervalMs),
			)
		} else {
			l.Error("timeout reached, will retry",
				slog.Int("retry_number", int(evtProcessingState.RetryNumber)),
				slog.Int("max_retries", int(evtProcessingState.ConsumerOptionMaxRetries)),
				slog.Int64("retry_interval_ms", evtProcessingState.ConsumerOptionRetryIntervalMs),
			)

			newEvtProcessingState, _ := evtProcessingState.Next()

			newEvtProcessingState.ProcessableAt = p.nextProcessableAt(evtProcessingState)
			newEvtProcessingState.RetryNumber += 1

			nextEvtProcessingStateResult = append(nextEvtProcessingStateResult, newEvtProcessingState)
		}

		err = p.store.SaveEventHandlers(ctx, append(nextEvtProcessingStateResult, evtProcessingState))
		if err != nil {
			l.Error("failed to save event consumer", kitslog.Err(err))
		}
	}
}

func (p PostgresEventStore) processConsumer(
	ctx context.Context,
	evt kitevent.Event,
	consumer kitevent.Consumer,
	evtProcessingState *EventProcessingState,
) {
	startHandlerAt := time.Now()

	l := p.logger.With(
		slog.Int("event_id", int(evtProcessingState.EventID)),
		slog.String("consumer", consumer.Name()),
		slog.String("event_name", evt.EventName().Name))

	l.Info("processing event")

	var err error
	chErr := wrapResultAsChanErr(func() error {
		return kitevent.CallConsumer(kitevent.CallConsumerParams{
			Ctx:     ctx,
			Event:   evt,
			Handler: consumer,
		})
	})

	ctx, cancel := context.WithTimeout(ctx, time.Duration(evtProcessingState.ConsumerOptionTimeoutMs)*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		err = ctx.Err()
		break
	case cerr := <-chErr:
		err = cerr
		break
	}

	var (
		nextEvtProcessingStateResult []*EventProcessingState
	)

	evtProcessingState.UpdatedAt = pgutils.TimestampUTC(time.Now())
	evtProcessingState.RunAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))
	evtProcessingState.DurationMs = time.Since(startHandlerAt).Milliseconds()

	if errors.Is(err, context.DeadlineExceeded) {
		// should be handled by monitorTimeoutEvents
		return
	}

	if err != nil {
		evtProcessingState.Error = lo.ToPtr(err.Error())
		evtProcessingState.Status = EventProcessingStateStatusFailed
		evtProcessingState.FailedAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))

		if evtProcessingState.RetryNumber >= evtProcessingState.ConsumerOptionMaxRetries {
			l.Error("failed to process event and max retries reached", kitslog.Err(err),
				slog.Int("retry_number", int(evtProcessingState.RetryNumber)),
				slog.Int("max_retries", int(evtProcessingState.ConsumerOptionMaxRetries)),
				slog.Int64("retry_interval_ms", evtProcessingState.ConsumerOptionRetryIntervalMs),
			)
		} else {
			l.Error("failed to process event, will retry", kitslog.Err(err),
				slog.Int("retry_number", int(evtProcessingState.RetryNumber)),
				slog.Int("max_retries", int(evtProcessingState.ConsumerOptionMaxRetries)),
				slog.Int64("retry_interval_ms", evtProcessingState.ConsumerOptionRetryIntervalMs),
			)

			newEvtProcessingState, _ := evtProcessingState.Next()

			newEvtProcessingState.ProcessableAt = p.nextProcessableAt(evtProcessingState)
			newEvtProcessingState.RetryNumber += 1

			nextEvtProcessingStateResult = append(nextEvtProcessingStateResult, newEvtProcessingState)
		}

	} else {
		evtProcessingState.Status = EventProcessingStateStatusSuccess
		evtProcessingState.SuccessAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))
	}

	err = p.store.SaveEventHandlers(ctx, append(nextEvtProcessingStateResult, evtProcessingState))
	if err != nil {
		l.Error("failed to save event consumer", kitslog.Err(err))
	}
}

func (p PostgresEventStore) nextProcessableAt(evtProcessingState *EventProcessingState) pgtype.Timestamp {
	return pgutils.TimestampUTC(time.Now().Add(
		time.Duration(evtProcessingState.ConsumerOptionRetryIntervalMs) * time.Millisecond))
}

func (p PostgresEventStore) nextEvent(ctx context.Context) (*EventProcessingState, error) {
	ctx, cancelCtx := context.WithTimeout(ctx, time.Second)
	defer cancelCtx()

	evtHandlerResult, err := p.store.FindAvailableEvent(ctx)
	if err != nil {
		return nil, err
	}

	return evtHandlerResult, nil
}

func (p PostgresEventStore) nextEventInTimeout(ctx context.Context) (*EventProcessingState, error) {
	ctx, cancelCtx := context.WithTimeout(ctx, time.Second)
	defer cancelCtx()

	evtHandlerResult, err := p.store.FindPendingTimeoutEvent(ctx)
	if err != nil {
		return nil, err
	}

	return evtHandlerResult, nil
}

func wrapResultAsChanErr(f func() error) chan error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- f()
	}()
	return errChan
}
