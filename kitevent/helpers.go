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
	handleFunc := reflect.ValueOf(handler).MethodByName("Handle")
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

func PayloadToEvent(handler Handler, evt []byte) (Event, error) {
	val := reflect.New(reflect.ValueOf(handler).
		MethodByName("Handle").
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

type CallHandlerParams struct {
	Ctx     context.Context
	Event   Event
	Handler Handler
}

func CallHandler(p CallHandlerParams) error {
	handleFunc := reflect.ValueOf(p.Handler).MethodByName("Handle")
	ret := handleFunc.Call([]reflect.Value{reflect.ValueOf(p.Ctx), reflect.ValueOf(p.Event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		return ret[0].Interface().(error)
	}

	return nil
}

type LocalCallHandlerParams struct {
	Ctx           context.Context
	Event         Event
	Producer      Producer
	Opts          *ProducerOptions
	Handler       Handler
	Logger        *slog.Logger
	IsProduceSync bool
}

func LocalCallHandler(p LocalCallHandlerParams) error {
	produceAgain := func() error {
		if p.Handler.Options().RetryInterval != nil {
			p.Opts.WithProduceAt(time.Now().Add(*p.Handler.Options().RetryInterval))
		}

		if p.IsProduceSync {
			return p.Producer.ProduceSync(p.Ctx, p.Event, p.Opts)
		} else {
			return p.Producer.Produce(p.Ctx, p.Event, p.Opts)
		}
	}

	handleFunc := reflect.ValueOf(p.Handler).MethodByName("Handle")
	ret := handleFunc.Call([]reflect.Value{reflect.ValueOf(p.Ctx), reflect.ValueOf(p.Event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		err := ret[0].Interface().(error)

		sl := slog.With(kitslog.Err(err), slog.String("event_name", p.Event.EventName().Name))

		if err != nil && p.Handler.Options().MaxRetries != nil {
			maxRetry := *p.Handler.Options().MaxRetries
			retryCount := p.Opts.RetryCount

			if retryCount < maxRetry {
				sl.Error("will retry Event because Handler gets an error",
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
