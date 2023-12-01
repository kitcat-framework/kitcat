package kiteventpg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/expectedsh/kitcat/kitevent"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/expectedsh/kitcat/pkg/kitpg/pgutils"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"log/slog"
	"time"
)

type PostgresEventStoreConfig struct {
	DelayBetweenPeek time.Duration `env:"KIT_EVENT_PG_DELAY_BETWEEN_PEEK" envDefault:"500ms"`
	CreateSchema     bool          `env:"KIT_EVENT_PG_CREATE_SCHEMA" envDefault:"true"`
}

type PostgresEventStore struct {
	handlers   map[kitevent.EventName][]kitevent.Handler
	db         *gorm.DB
	store      EventStoreStorage
	logger     *slog.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc

	config *PostgresEventStoreConfig
}

func New(db *gorm.DB, logger *slog.Logger, config PostgresEventStoreConfig) *PostgresEventStore {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &PostgresEventStore{
		db: db,
		logger: logger.With(
			kitslog.Module("kitevent"),
			slog.String("store", "postgres")),
		handlers:   make(map[kitevent.EventName][]kitevent.Handler),
		store:      NewPgEventStore(db),
		ctx:        ctx,
		cancelFunc: cancelFunc,
		config:     &config,
	}
}

func (p PostgresEventStore) Produce(ctx context.Context, event kitevent.Event, opt *kitevent.ProducerOptions) error {
	handlersConcerned := p.handlers[event.EventName()]
	if len(handlersConcerned) == 0 {
		return errors.New("no handler found for event")
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

	processors := make([]*HandlerResult, len(handlersConcerned))
	for i, handler := range handlersConcerned {
		maxRetries := int32(1)
		if handler.Options().MaxRetries != nil {
			maxRetries = *handler.Options().MaxRetries
		}

		processableAt := time.Now()
		if opt.ProduceAt != nil {
			processableAt = *opt.ProduceAt
		}

		retryIntervalMs := int64(0)
		if handler.Options().RetryInterval != nil {
			retryIntervalMs = handler.Options().RetryInterval.Milliseconds()
		}

		processors[i] = &HandlerResult{
			HandlerName:       handler.Name(),
			Status:            EventHandlerResultStatusTakeable,
			Error:             nil,
			RetryNumber:       1,
			MaxRetries:        maxRetries,
			RetryIntervalMs:   retryIntervalMs,
			CreatedAt:         pgutils.TimestampUTC(time.Now()),
			UpdatedAt:         pgutils.TimestampUTC(time.Now()),
			ProcessableAt:     pgutils.TimestampUTC(processableAt),
			RunAt:             nil,
			HandlerDurationMs: 0,
			FailedAt:          nil,
			SuccessAt:         nil,
			PendingAt:         nil,
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
		return kitevent.LocalCallHandler(kitevent.LocalCallHandlerParams{
			Ctx:           ctx,
			Event:         event,
			Producer:      p,
			Opts:          opts,
			Handler:       handler,
			Logger:        p.logger,
			IsProduceSync: true,
		})
	}

	return nil
}

func (p PostgresEventStore) AddEventHandler(eventName kitevent.EventName, handler kitevent.Handler) {
	p.handlers[eventName] = append(p.handlers[eventName], handler)
}

func (p PostgresEventStore) OnStart(ctx context.Context) error {
	if p.config.CreateSchema {
		err := p.db.Exec("CREATE SCHEMA IF NOT EXISTS kitevent;").Error
		if err != nil {
			return fmt.Errorf("failed to create kitevent schema: %w", err)
		}
	}

	err := p.db.WithContext(ctx).AutoMigrate(&Event{}, &HandlerResult{})
	if err != nil {
		return fmt.Errorf("failed to migrate event model: %w", err)
	}

	go p.Run(p.ctx)

	return nil
}

func (p PostgresEventStore) OnStop(_ context.Context) error {
	p.cancelFunc()

	return nil
}

func (p PostgresEventStore) Name() string {
	return "postgres"
}

// Run is a blocking function that will loop over handler results to find those who are in TAKEABLE status.
// If one is found, it will call the handler associated to the event and update the handler result accordingly.
// If none is found, it will wait for 500ms and try again.
func (p PostgresEventStore) Run(ctx context.Context) {
	for {
		eventHandlerResult, err := p.getHandlerResult(ctx)
		if err != nil {
			time.Sleep(time.Millisecond * 500) // TODO: make it configurable
			continue
		}

		handlers := p.handlers[kitevent.NewEventName(eventHandlerResult.Event.EventName)]
		if len(handlers) == 0 {
			continue
		}

		for _, handler := range handlers {
			if eventHandlerResult.HandlerName != handler.Name() {
				continue
			}

			event, err := kitevent.PayloadToEvent(handler, eventHandlerResult.Event.Payload)
			if err != nil {
				p.logger.Error("failed to convert payload to event", kitslog.Err(err))
				continue
			}

			// TODO: process them in a pool to avoid having lot of PENDING events whereas
			// the handler is not able to process them all
			go p.processHandlerResult(
				ctx,
				event,
				handler,
				eventHandlerResult,
			)

			break
		}
	}
}

func (p PostgresEventStore) processHandlerResult(
	ctx context.Context,
	evt kitevent.Event,
	handler kitevent.Handler,
	evtHandlerResult *HandlerResult,
) {
	startHandlerAt := time.Now()

	l := p.logger.With(
		slog.Int("event_id", int(evtHandlerResult.EventID)),
		slog.String("handler", handler.Name()),
		slog.String("event", string(evtHandlerResult.Event.Payload)),
		slog.String("event_name", evt.EventName().Name))

	l.Info("processing event")
	err := kitevent.CallHandler(kitevent.CallHandlerParams{
		Ctx:     ctx,
		Event:   evt,
		Handler: handler,
	})

	var (
		nextEvtHandlerResult []*HandlerResult
	)

	evtHandlerResult.UpdatedAt = pgutils.TimestampUTC(time.Now())
	evtHandlerResult.RunAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))
	evtHandlerResult.HandlerDurationMs = time.Since(startHandlerAt).Milliseconds()

	if err != nil {

		evtHandlerResult.Error = lo.ToPtr(err.Error())

		if evtHandlerResult.RetryNumber >= evtHandlerResult.MaxRetries {
			l.Error("failed to process event and max retries reached", kitslog.Err(err),
				slog.Int("retry_number", int(evtHandlerResult.RetryNumber)),
				slog.Int("max_retries", int(evtHandlerResult.MaxRetries)),
				slog.Int64("retry_interval_ms", evtHandlerResult.RetryIntervalMs),
			)

			evtHandlerResult.Status = EventHandlerResultStatusFailed
			evtHandlerResult.FailedAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))

		} else {
			l.Error("failed to process event, will retry", kitslog.Err(err),
				slog.Int("retry_number", int(evtHandlerResult.RetryNumber)),
				slog.Int("max_retries", int(evtHandlerResult.MaxRetries)),
				slog.Int64("retry_interval_ms", evtHandlerResult.RetryIntervalMs),
			)

			nextEvtHandlerResult = append(nextEvtHandlerResult, &HandlerResult{
				HandlerName:       evtHandlerResult.HandlerName,
				EventID:           evtHandlerResult.EventID,
				Event:             evtHandlerResult.Event,
				Status:            EventHandlerResultStatusTakeable,
				RetryNumber:       evtHandlerResult.RetryNumber + 1,
				MaxRetries:        evtHandlerResult.MaxRetries,
				RetryIntervalMs:   evtHandlerResult.RetryIntervalMs,
				CreatedAt:         pgutils.TimestampUTC(time.Now()),
				UpdatedAt:         pgutils.TimestampUTC(time.Now()),
				ProcessableAt:     pgutils.TimestampUTC(time.Now().Add(time.Duration(evtHandlerResult.RetryIntervalMs) * time.Millisecond)),
				HandlerDurationMs: 0,
				FailedAt:          nil,
				SuccessAt:         nil,
			})
		}

	} else {
		evtHandlerResult.Status = EventHandlerResultStatusSuccess
		evtHandlerResult.SuccessAt = lo.ToPtr(pgutils.TimestampUTC(time.Now()))
	}

	err = p.store.SaveEventHandlers(ctx, append(nextEvtHandlerResult, evtHandlerResult))
	if err != nil {
		l.Error("failed to save event handler", kitslog.Err(err))
	}
}

func (p PostgresEventStore) getHandlerResult(ctx context.Context) (*HandlerResult, error) {
	ctx, cancelCtx := context.WithTimeout(ctx, time.Second)
	defer cancelCtx()

	evtHandlerResult, err := p.store.PeekEventHandlerResult(ctx)
	if err != nil {
		return nil, err
	}

	return evtHandlerResult, nil
}
