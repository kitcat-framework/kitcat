package kiteventpg

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type EventStoreStorage interface {
	AddEvent(ctx context.Context, event Event, processor []*EventProcessingState) error
	FindAvailableEvent(ctx context.Context) (*EventProcessingState, error)
	FindPendingTimeoutEvent(ctx context.Context) (*EventProcessingState, error)
	SaveEventHandlers(ctx context.Context, handler []*EventProcessingState) error
}

type PgEventStore struct {
	db *gorm.DB
}

func NewPgEventStore(db *gorm.DB) *PgEventStore {
	return &PgEventStore{db: db}
}

func (p PgEventStore) AddEvent(ctx context.Context, event Event, processors []*EventProcessingState) error {
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

func (p PgEventStore) SaveEventHandlers(ctx context.Context, handlers []*EventProcessingState) error {
	err := p.db.WithContext(ctx).Model(handlers).Save(handlers).Error
	if err != nil {
		return fmt.Errorf("failed to update event handlers: %w", err)
	}

	return nil
}

func (p PgEventStore) FindAvailableEvent(ctx context.Context) (*EventProcessingState, error) {
	tx := p.db.Session(&gorm.Session{PrepareStmt: false, Context: ctx})
	const query = `
		update kitevent.event_processing_states
		set status = ?, pending_at = now() at time zone 'utc'
		where id = (
		  select id
		  from kitevent.event_processing_states
		  where status = ?
			and processable_at <= now() at time zone 'utc'
		  order by id
		  for update skip locked
		  limit 1
		)
		returning *;
	`

	var (
		handler EventProcessingState
		evt     Event
	)

	err := tx.Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(query, EventProcessingStateStatusPending,
			EventProcessingStateStatusAvailable).Scan(&handler).Error
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

func (p PgEventStore) FindPendingTimeoutEvent(ctx context.Context) (*EventProcessingState, error) {
	tx := p.db.Session(&gorm.Session{PrepareStmt: false, Context: ctx})
	const query = `
		update kitevent.event_processing_states
		set status = ?, 
		    updated_at = now() at time zone 'utc'
		where id = (
		  select id
		  from kitevent.event_processing_states
		  where status = ? and timeout_at <= now() at time zone 'utc'
		  order by id
		  for update skip locked
		  limit 1
		)
		returning *;
	`

	var (
		handler EventProcessingState
		evt     Event
	)

	err := tx.Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(query, EventProcessingStateStatusFailed, EventProcessingStateStatusPending).Scan(&handler).Error
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

//func UpdateTimeoutEvents(ctx context.Context, db *gorm.DB) error {
//	const query = `
//		update kitevent.event_processing_states
//		set status = ?
//		where status = ?
//			and processable_at <= now()
//	`
//
//	err := db.WithContext(ctx).Exec(query, EventProcessingStateStatusTimeout, EventProcessingStateStatusPending).Error
//	if err != nil {
//		return fmt.Errorf("failed to update timeout events: %w", err)
//	}
//
//	return nil
//}
