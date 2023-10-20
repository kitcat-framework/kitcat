package kitevent

import (
	"context"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"time"
)

type (
	// EventName is the name of the event
	EventName struct {
		Name string
	}

	Event interface {
		EventName() EventName
	}

	ProduceOptions struct {
		// MaxRetry is the maximum number of retry for an event
		// The implementation may provide a dead letter queue to store the event somewhere
		MaxRetry *int

		// ProduceAt is the time at which the event should be produced
		// If the time is in the past, the event will be produced immediately
		// If the time is in the future, the event will be produced at least after the
		// specified time not before
		ProduceAt *time.Time
	}

	ProduceOptionFn func(*ProduceOptions)

	Producer interface {
		// Produce is used to produce an event
		// The event will be consumed asynchronously by the implementation
		//
		// An error is returned if the event cannot be produced
		Produce(event Event, opts ...ProduceOptionFn) error

		// ProduceNow is used to produce an event immediately, instead of using the queue
		// The implementation must call the function synchronously
		//
		// An error is returned if one of the handler returns an error
		ProduceNow(ctx context.Context, event Event) error
	}

	Handler[T Event] interface {
		Handle(context.Context, T) error
		kitcat.Nameable
	}

	Store interface {
		Producer
		AddEventHandler(eventName EventName, handler kitcat.Nameable)
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
		Handlers []kitcat.Nameable `group:"kitevent.handler"`
	}
)

func ProduceOptionMaxRetry(maxRetry int) ProduceOptionFn {
	return func(option *ProduceOptions) {
		option.MaxRetry = &maxRetry
	}
}

func ProduceOptionAt(produceAt time.Time) ProduceOptionFn {
	return func(option *ProduceOptions) {
		option.ProduceAt = &produceAt
	}
}

func NewEventName(name string) EventName {
	return EventName{
		Name: name,
	}
}

func EventHandlerAnnotation(handler any) *kitdi.Annotation {
	return kitdi.Annotate(handler, kitdi.Group("kitevent.handler"), kitdi.As(new(kitcat.Nameable)))
}

func StoreAnnotation(store any) *kitdi.Annotation {
	return kitdi.Annotate(store, kitdi.Group("kitevent.store"), kitdi.As(new(Store)))
}
