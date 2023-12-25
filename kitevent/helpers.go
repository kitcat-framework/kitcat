package kitevent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitreflect"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"reflect"
	"time"
)

func IsHandler(handler kitcat.Nameable) bool {
	handleFunc := reflect.ValueOf(handler).MethodByName("Consume")
	if handleFunc.Kind() != reflect.Func {
		return false
	}

	if !kitreflect.EnsureInOutLength(handleFunc.Type(), 2, 1) {
		return false
	}

	if !kitreflect.EnsureInIsContext(handleFunc.Type()) {
		return false
	}

	if !kitreflect.EnsureOutIsError(handleFunc.Type()) {
		return false
	}

	if handleFunc.Type().In(1).Kind() != reflect.Ptr {
		return false
	}

	if !handleFunc.Type().In(1).AssignableTo(reflect.TypeOf((*Event)(nil)).Elem()) {
		return false
	}

	return true
}

func PayloadToEvent(handler Consumer, evt []byte) (Event, error) {
	val := reflect.New(reflect.ValueOf(handler).
		MethodByName("Consume").
		Type().In(1).Elem())

	err := json.Unmarshal(evt, val.Interface())
	if err != nil {
		return nil, err
	}

	event, ok := val.Interface().(Event)
	if !ok {
		return nil, errors.New("invalid event")
	}

	return event, nil

}

type CallConsumerParams struct {
	Ctx     context.Context
	Event   Event
	Handler Consumer
}

func CallConsumer(p CallConsumerParams) error {
	consumerFunc := reflect.ValueOf(p.Handler).MethodByName("Consume")
	ret := consumerFunc.Call([]reflect.Value{reflect.ValueOf(p.Ctx), reflect.ValueOf(p.Event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		return ret[0].Interface().(error)
	}

	return nil
}

type LocalCallConsumerParams struct {
	Ctx           context.Context
	Event         Event
	Producer      Producer
	Opts          *ProducerOptions
	Consumer      Consumer
	Logger        *slog.Logger
	IsProduceSync bool
}

func LocalCallHandler(p LocalCallConsumerParams) error {
	produceAgain := func() error {
		if p.Consumer.Options().RetryInterval != nil {
			p.Opts.WithProduceAt(time.Now().Add(*p.Consumer.Options().RetryInterval))
		}

		if p.IsProduceSync {
			return p.Producer.ProduceSync(p.Ctx, p.Event, p.Opts)
		} else {
			return p.Producer.Produce(p.Ctx, p.Event, p.Opts)
		}
	}

	handleFunc := reflect.ValueOf(p.Consumer).MethodByName("Consume")
	ret := handleFunc.Call([]reflect.Value{reflect.ValueOf(p.Ctx), reflect.ValueOf(p.Event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		err := ret[0].Interface().(error)

		sl := slog.With(kitslog.Err(err), slog.String("event_name", p.Event.EventName().Name))

		if err != nil && p.Consumer.Options().MaxRetries != nil {
			maxRetry := *p.Consumer.Options().MaxRetries
			retryCount := p.Opts.RetryCount

			if retryCount < maxRetry {
				sl.Error("will retry Event because Consumer gets an error",
					slog.Int("current_retry_count", int(retryCount)),
					slog.Int("max_retry", int(maxRetry)),
				)

				p.Opts.RetryCount += 1

				return produceAgain()
			} else {
				sl.Error("unable to execute Event, reached max retry",
					slog.Int("retry_count", int(retryCount)),
					slog.Int("max_retry", int(maxRetry)),
				)
				return err
			}
		}

		sl.Error("an error occurred")

		return err
	}

	return nil
}
