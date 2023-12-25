package kiteventpg

import (
	"dario.cat/mergo"
	"fmt"
	"github.com/expectedsh/kitcat/pkg/kitpg/pgutils"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/datatypes"
	"time"
)

type Event struct {
	ID      int32          `gorm:"primaryKey"`
	Payload datatypes.JSON `gorm:"type:jsonb"`

	EventName string `gorm:"type:varchar(255)"`

	CreatedAt pgtype.Timestamp `gorm:"type:timestamp"`
	UpdatedAt pgtype.Timestamp `gorm:"type:timestamp"`
}

func (Event) TableName() string {
	return "kitevent.events"
}

type EventProcessingStateStatus string

const (
	EventProcessingStateStatusFailed    EventProcessingStateStatus = "FAILED"
	EventProcessingStateStatusSuccess   EventProcessingStateStatus = "SUCCESS"
	EventProcessingStateStatusPending   EventProcessingStateStatus = "PENDING"
	EventProcessingStateStatusAvailable EventProcessingStateStatus = "AVAILABLE"
)

type EventProcessingState struct {
	ID           int32
	ConsumerName string

	EventID int32
	Event   *Event

	Status EventProcessingStateStatus `gorm:"index"`
	Error  *string

	ConsumerOptionMaxRetries      int32
	ConsumerOptionRetryIntervalMs int64
	ConsumerOptionTimeoutMs       int64

	CreatedAt     pgtype.Timestamp `gorm:"type:timestamp"`
	UpdatedAt     pgtype.Timestamp `gorm:"type:timestamp"`
	ProcessableAt pgtype.Timestamp `gorm:"type:timestamp"`

	RunAt       *pgtype.Timestamp `gorm:"type:timestamp"`
	RetryNumber int32
	DurationMs  int64
	TimeoutAt   *pgtype.Timestamp `gorm:"->;type:timestamp GENERATED ALWAYS AS (coalesce(pending_at, created_at) + (consumer_option_timeout_ms * interval '1 millisecond')) STORED;now();index"`

	FailedAt  *pgtype.Timestamp `gorm:"type:timestamp"`
	SuccessAt *pgtype.Timestamp `gorm:"type:timestamp"`
	PendingAt *pgtype.Timestamp `gorm:"type:timestamp"`
}

func (EventProcessingState) TableName() string {
	return "kitevent.event_processing_states"
}

// Next returns a new EventProcessingState with the same values as the current one
// except for the following fields:
// - Status: set to AVAILABLE
// - Error: set to nil
// - CreatedAt: set to the current time
// - UpdatedAt: set to the current time
// - ProcessableAt: set to the current time
// - RunAt: set to nil
// - RetryNumber: set to 0
// - DurationMs: set to 0
// - TimeoutAt: set to nil
// - FailedAt: set to nil
// - SuccessAt: set to nil
// - PendingAt: set to nil
func (c *EventProcessingState) Next() (*EventProcessingState, error) {
	dest := EventProcessingState{}
	err := mergo.Merge(&dest, c)
	if err != nil {
		return nil, fmt.Errorf("failed to clone event processing state: %w", err)
	}

	dest.ID = 0
	dest.Status = EventProcessingStateStatusAvailable
	dest.Error = nil
	dest.CreatedAt = pgutils.TimestampUTC(time.Now())
	dest.UpdatedAt = pgutils.TimestampUTC(time.Now())
	dest.ProcessableAt = pgutils.TimestampUTC(time.Now())
	dest.RunAt = nil
	dest.DurationMs = 0
	dest.TimeoutAt = nil
	dest.FailedAt = nil
	dest.SuccessAt = nil
	dest.PendingAt = nil

	return &dest, nil
}
