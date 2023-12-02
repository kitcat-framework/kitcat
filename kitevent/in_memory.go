package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"time"
)

type InMemoryEventStore struct {
	handlers map[EventName][]Consumer
	logger   *slog.Logger
}

func NewInMemoryEventStore(logger *slog.Logger) *InMemoryEventStore {
	return &InMemoryEventStore{
		handlers: make(map[EventName][]Consumer),
		logger: logger.With(
			kitslog.Module("kitevent"),
			slog.String("store", "in-memory")),
	}
}

func (p *InMemoryEventStore) AddConsumer(eventName EventName, listener Consumer) {
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
			_ = LocalCallHandler(LocalCallConsumerParams{
				Ctx:           ctx,
				Event:         event,
				Producer:      p,
				Opts:          opts,
				Consumer:      handler,
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
		return LocalCallHandler(LocalCallConsumerParams{
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

func (p *InMemoryEventStore) Name() string {
	return "in-memory"
}

func (p *InMemoryEventStore) OnStart(_ context.Context) error {
	return nil
}

func (p *InMemoryEventStore) OnStop(_ context.Context) error {
	return nil
}
