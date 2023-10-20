package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"time"
)

type InMemoryEventStore struct {
	listeners map[EventName][]kitcat.Nameable
	logger    *slog.Logger
}

func NewInMemoryEventStore(logger *slog.Logger) *InMemoryEventStore {
	return &InMemoryEventStore{
		listeners: make(map[EventName][]kitcat.Nameable),
		logger: logger.With(
			kitslog.Module("kitevent"),
			slog.String("store", "in-memory")),
	}
}

func (p *InMemoryEventStore) AddEventHandler(eventName EventName, listener kitcat.Nameable) {
	slog.Info("add event handler", slog.String("event_name", eventName.Name))
	p.listeners[eventName] = append(p.listeners[eventName], listener)
}

func (p *InMemoryEventStore) Produce(event Event, opts ...ProduceOptionFn) error {
	options := &ProduceOptions{}
	for _, opt := range opts {
		opt(options)
	}

	go func() {
		if options != nil && options.ProduceAt.After(time.Now()) {
			time.Sleep(time.Until(*options.ProduceAt))
		}
		retryCount := 0
		for {
			err := p.ProduceNow(context.Background(), event)
			if err != nil {
				p.logger.Error("unable to execute event",
					kitslog.Err(err),
					slog.String("event_name", event.EventName().Name),
					slog.Int("retry_count", retryCount))
			}

			if options != nil && options.MaxRetry != nil {
				if retryCount >= *options.MaxRetry {
					return
				}
				retryCount++
			} else {
				return
			}
		}
	}()

	return nil
}

func (p *InMemoryEventStore) ProduceNow(ctx context.Context, event Event) error {
	listeners, ok := p.listeners[event.EventName()]

	if !ok {
		return nil
	}

	for _, listener := range listeners {
		return CallHandler(ctx, listener, event)
	}

	return nil
}

func (p *InMemoryEventStore) Name() string {
	return "in-memory"
}

func (p *InMemoryEventStore) OnStart() error {
	return nil
}

func (p *InMemoryEventStore) OnStop() error {
	return nil
}
