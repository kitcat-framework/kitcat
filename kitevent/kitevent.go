package kitevent

import (
	"context"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
)

type (
	// EventName is the name of the event
	EventName struct {
		Name string
	}

	Event interface {
		EventName() EventName
	}

	Producer interface {
		Produce(event Event) error
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
