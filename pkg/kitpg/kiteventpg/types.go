package kiteventpg

import (
	"dario.cat/mergo"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kitcat-framework/kitcat/pkg/kitpg/pgutils"
	"gorm.io/datatypes"
	"time"
)

type Event struct {
	ID      int32
	Payload datatypes.JSON

	EventName string

	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
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

	Status EventProcessingStateStatus
	Error  *string

	ConsumerOptionMaxRetries      int32
	ConsumerOptionRetryIntervalMs int64
	ConsumerOptionTimeoutMs       int64

	CreatedAt     pgtype.Timestamp
	UpdatedAt     pgtype.Timestamp
	ProcessableAt pgtype.Timestamp

	RunAt       *pgtype.Timestamp
	RetryNumber int32
	DurationMs  int64
	TimeoutAt   *pgtype.Timestamp

	FailedAt  *pgtype.Timestamp
	SuccessAt *pgtype.Timestamp
	PendingAt *pgtype.Timestamp
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
