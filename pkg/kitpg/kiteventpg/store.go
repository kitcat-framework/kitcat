package kiteventpg

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type EventStoreStorage interface {
	AddEvent(ctx context.Context, event Event, processor []*HandlerResult) error
	PeekEventHandlerResult(ctx context.Context) (*HandlerResult, error)
	SaveEventHandlers(ctx context.Context, handler []*HandlerResult) error
}

type PgEventStore struct {
	db *gorm.DB
}

func NewPgEventStore(db *gorm.DB) *PgEventStore {
	return &PgEventStore{db: db}
}

func (p PgEventStore) AddEvent(ctx context.Context, event Event, processors []*HandlerResult) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		for _, processor := range processors {
			processor.EventID = event.ID
		}

		err := tx.Create(processors).Error
		if err != nil {
			return fmt.Errorf("failed to create event processors: %w", err)
		}

		return nil
	})
}

func (p PgEventStore) SaveEventHandlers(ctx context.Context, handlers []*HandlerResult) error {
	err := p.db.WithContext(ctx).Model(handlers).Save(handlers).Error
	if err != nil {
		return fmt.Errorf("failed to update event handlers: %w", err)
	}

	return nil
}

func (p PgEventStore) PeekEventHandlerResult(ctx context.Context) (*HandlerResult, error) {
	tx := p.db.Session(&gorm.Session{PrepareStmt: false, Context: ctx})
	const query = `
		update kitevent.handler_results
		set status = 'PENDING', pending_at = now()
		where id = (
		  select id
		  from kitevent.handler_results
		  where status = 'TAKEABLE'
			and processable_at <= now()
		  order by id
		  for update skip locked
		  limit 1
		)
		returning *;
	`

	var (
		handler HandlerResult
		evt     Event
	)

	err := tx.Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(query).Scan(&handler).Error
		if err != nil {
			return fmt.Errorf("failed to get event handler: %w", err)
		}

		if handler.ID == 0 {
			return gorm.ErrRecordNotFound
		}

		err = tx.First(&evt, handler.EventID).Error
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	handler.Event = &evt

	return &handler, nil
}
