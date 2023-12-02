package kiteventpg

import (
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/datatypes"
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
	TimeoutAt   *pgtype.Timestamp `gorm:"->;type:timestamp GENERATED ALWAYS AS (run_at + (consumer_option_timeout_ms * interval '1 millisecond')) STORED;default:dummy();index"`

	FailedAt  *pgtype.Timestamp `gorm:"type:timestamp"`
	SuccessAt *pgtype.Timestamp `gorm:"type:timestamp"`
	PendingAt *pgtype.Timestamp `gorm:"type:timestamp"`
}

func (EventProcessingState) TableName() string {
	return "kitevent.event_processing_states"
}
