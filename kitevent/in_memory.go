package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
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

func (p *InMemoryEventStore) Produce(event Event) error {
	return p.ProduceNow(context.Background(), event)
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
