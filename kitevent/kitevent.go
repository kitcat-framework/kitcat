package kitevent

import (
	"context"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/google/uuid"
	"time"
)

type (
	// EventName is the name of the event
	EventName struct {
		Name string
	}

	// Event is the interface that must be implemented by an event
	Event interface {
		EventName() EventName
	}

	// HandlerOptions is the options for an event handler
	HandlerOptions struct {
		// MaxRetry is the maximum number of retry for an event
		// The implementation may provide a dead letter queue to store the event somewhere
		MaxRetry *int

		// RetryInterval is the interval between each retry
		RetryInterval *time.Duration
	}

	// ProducerOptions is the options for an event producer
	ProducerOptions struct {
		// ProduceAt is the time at which the event should be produced
		// If the time is in the past, the event will be produced immediately
		// If the time is in the future, the event will be produced at least after the
		// specified time not before
		//
		// This option is generally ignored by Producer.ProduceSync
		ProduceAt *time.Time

		// Keep track of the number of retry for this particular event
		RetryCount int

		// Metadata is the metadata of the event
		Metadata map[string]any
	}

	// Producer is used to produce an event
	Producer interface {
		// Produce is used to produce an event
		// The event will be consumed asynchronously by the implementation
		//
		// An error is returned if the event cannot be produced
		Produce(ctx context.Context, event Event, opt *ProducerOptions) error

		// ProduceSync is used to produce an event synchronously, instead of using the queue
		// The implementation must call the function synchronously
		//
		// An error is returned if one of the handler returns an error
		ProduceSync(ctx context.Context, event Event, opt *ProducerOptions) error
	}

	Handler interface {
		Options() *HandlerOptions
		kitcat.Nameable
	}

	HandlerFunc[T Event] interface {
		Handle(ctx context.Context, event T) error
	}

	Store interface {
		Producer
		AddEventHandler(eventName EventName, handler Handler)
		OnStart() error
		OnStop() error
		kitcat.Nameable
	}

	stores struct {
		dig.In
		Stores []Store `group:"kitevent.store"`
	}

	handlers struct {
		dig.In
		Handlers []Handler `group:"kitevent.handler"`
	}
)

func NewHandlerOptions() *HandlerOptions {
	return &HandlerOptions{}
}

func (h *HandlerOptions) WithMaxRetry(maxRetry int) *HandlerOptions {
	h.MaxRetry = &maxRetry
	return h
}

func (h *HandlerOptions) WithRetryInterval(retryInterval time.Duration) *HandlerOptions {
	h.RetryInterval = &retryInterval
	return h
}

func NewProducerOptions() *ProducerOptions {
	return &ProducerOptions{
		Metadata: map[string]any{
			"id": uuid.New().String(),
		},
	}
}

func (p *ProducerOptions) WithMetadata(key string, value any) *ProducerOptions {
	p.Metadata[key] = value
	return p
}

func (p *ProducerOptions) WithProduceAt(produceAt time.Time) *ProducerOptions {
	p.ProduceAt = &produceAt
	return p
}

func (p *ProducerOptions) WithAddRetryCount() *ProducerOptions {
	p.RetryCount += 1
	return p
}

func NewEventName(name string) EventName {
	return EventName{
		Name: name,
	}
}

func EventHandlerAnnotation(handler any) *kitdi.Annotation {
	return kitdi.Annotate(handler, kitdi.Group("kitevent.handler"), kitdi.As(new(Handler)))
}

func StoreAnnotation(store any) *kitdi.Annotation {
	return kitdi.Annotate(store, kitdi.Group("kitevent.store"), kitdi.As(new(Store)))
}
