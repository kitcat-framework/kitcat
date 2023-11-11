package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"time"
)

type InMemoryEventStore struct {
	handlers map[EventName][]Handler
	logger   *slog.Logger
}

func NewInMemoryEventStore(logger *slog.Logger) *InMemoryEventStore {
	return &InMemoryEventStore{
		handlers: make(map[EventName][]Handler),
		logger: logger.With(
			kitslog.Module("kitevent"),
			slog.String("store", "in-memory")),
	}
}

func (p *InMemoryEventStore) AddEventHandler(eventName EventName, listener Handler) {
	slog.Info("add Event Handler", slog.String("event_name", eventName.Name))
	p.handlers[eventName] = append(p.handlers[eventName], listener)
}

func (p *InMemoryEventStore) Produce(ctx context.Context, event Event, opts *ProducerOptions) error {
	if opts == nil {
		opts = NewProducerOptions()
	}

	handlers, ok := p.handlers[event.EventName()]
	if !ok {
		return nil
	}

	go func() {
		if opts != nil && opts.ProduceAt.After(time.Now()) {
			time.Sleep(time.Until(*opts.ProduceAt))
		}

		for _, handler := range handlers {
			_ = LocalCallHandler(LocalCallHandlerParams{
				Ctx:           ctx,
				Event:         event,
				Producer:      p,
				Opts:          opts,
				Handler:       handler,
				Logger:        p.logger,
				IsProduceSync: false,
			})
		}
	}()

	return nil
}

func (p *InMemoryEventStore) ProduceSync(ctx context.Context, event Event, opts *ProducerOptions) error {
	if opts == nil {
		opts = NewProducerOptions()
	}

	handlers, ok := p.handlers[event.EventName()]

	if !ok {
		return nil
	}

	for _, handler := range handlers {
		return LocalCallHandler(LocalCallHandlerParams{
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

func (p *InMemoryEventStore) Name() string {
	return "in-memory"
}

func (p *InMemoryEventStore) OnStart(_ context.Context) error {
	return nil
}

func (p *InMemoryEventStore) OnStop(_ context.Context) error {
	return nil
}
