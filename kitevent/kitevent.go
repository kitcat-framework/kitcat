package kitevent

import (
	"context"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"time"
)

type (
	// EventName is the name of the Event
	EventName struct {
		Name string
	}

	// Event is the interface that must be implemented by an Event
	Event interface {
		EventName() EventName
	}

	// ConsumerOptions is the options for an Event Consumer
	ConsumerOptions struct {
		// MaxRetries is the maximum number of retry for an Event
		// The implementation may provide a dead letter queue to store the Event somewhere
		MaxRetries *int32

		// RetryInterval is the interval between each retry
		RetryInterval *time.Duration

		// The duration that the server will wait for an consumer for any individual event once it has been delivered.
		// If a consumer don't respond before the timeout, the event will be retried if the MaxRetries is not reached.
		ConsumeTimeout *time.Duration
	}

	// ProducerOptions is the options for an Event Producer
	ProducerOptions struct {
		// ProduceAt is the time at which the Event should be produced
		// If the time is in the past, the Event will be produced immediately
		// If the time is in the future, the Event will be produced at least after the
		// specified time not before
		//
		// This option is generally ignored by Producer.ProduceSync
		ProduceAt *time.Time

		// Keep track of the number of retry for this particular Event
		RetryCount int32

		// Metadata is the metadata of the Event
		Metadata map[string]any
	}

	// Producer is used to produce an Event
	Producer interface {
		// Produce is used to produce an Event
		// The Event will be consumed asynchronously by the implementation
		//
		// An error is returned if the Event cannot be produced
		Produce(ctx context.Context, event Event, opt *ProducerOptions) error

		// ProduceSync is used to produce an Event synchronously, instead of using the queue
		// The implementation must call the function synchronously
		//
		// An error is returned if one of the Consumer returns an error
		//
		// Generally to implement this method, you can use the LocalCallHandler function
		// You should not use the store to produce the Event.
		ProduceSync(ctx context.Context, event Event, opt *ProducerOptions) error
	}

	Consumer interface {
		Options() *ConsumerOptions
		kitcat.Nameable
	}

	ConsumerFunc[T Event] interface {
		Consume(ctx context.Context, event T) error
	}

	Store interface {
		Producer
		AddConsumer(eventName EventName, consumer Consumer)
		OnStart(ctx context.Context) error
		OnStop(ctx context.Context) error
		kitcat.Nameable
	}

	stores struct {
		dig.In
		Stores []Store `group:"kitevent.store"`
	}

	consumers struct {
		dig.In
		Consumers []Consumer `group:"kitevent.consumer"`
	}
)

func NewConsumerOptions() *ConsumerOptions {
	return &ConsumerOptions{
		ConsumeTimeout: lo.ToPtr(1 * time.Minute),
	}
}

func (h *ConsumerOptions) WithMaxRetry(maxRetry int32) *ConsumerOptions {
	h.MaxRetries = &maxRetry
	return h
}

func (h *ConsumerOptions) WithRetryInterval(retryInterval time.Duration) *ConsumerOptions {
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

func ProvideConsumer(consumer any) *kitdi.Annotation {
	return kitdi.Annotate(consumer, kitdi.Group("kitevent.consumer"), kitdi.As(new(Consumer)))
}

func ProvideStore(store any) *kitdi.Annotation {
	return kitdi.Annotate(store, kitdi.Group("kitevent.store"), kitdi.As(new(Store)))
}
