package kitevent

import (
	"context"
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

	if !handleFunc.Type().In(1).AssignableTo(reflect.TypeOf((*Event)(nil)).Elem()) {
		return false
	}

	return true
}

type CallHandlerParams struct {
	ctx           context.Context
	event         Event
	producer      Producer
	opts          *ProducerOptions
	handler       Handler
	logger        *slog.Logger
	isProduceSync bool
}

func CallHandler(p CallHandlerParams) error {
	produceAgain := func() error {
		if p.handler.Options().RetryInterval != nil {
			p.opts.WithProduceAt(time.Now().Add(*p.handler.Options().RetryInterval))
		}

		if p.isProduceSync {
			return p.producer.ProduceSync(p.ctx, p.event, p.opts)
		} else {
			return p.producer.Produce(p.ctx, p.event, p.opts)
		}
	}

	handleFunc := reflect.ValueOf(p.handler).MethodByName("Handle")
	ret := handleFunc.Call([]reflect.Value{reflect.ValueOf(p.ctx), reflect.ValueOf(p.event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		err := ret[0].Interface().(error)

		sl := slog.With(kitslog.Err(err), slog.String("event_name", p.event.EventName().Name))

		if err != nil && p.handler.Options().MaxRetry != nil {
			maxRetry := *p.handler.Options().MaxRetry
			retryCount := p.opts.RetryCount

			if retryCount < maxRetry {
				sl.Error("will retry event because handler gets an error",
					slog.Int("current_retry_count", retryCount),
					slog.Int("max_retry", maxRetry),
				)

				p.opts.RetryCount += 1

				return produceAgain()
			} else {
				sl.Error("unable to execute event, reached max retry",
					slog.Int("retry_count", retryCount),
					slog.Int("max_retry", maxRetry),
				)
				return err
			}
		}

		sl.Error("an error occured")

		return err
	}

	return nil
}
