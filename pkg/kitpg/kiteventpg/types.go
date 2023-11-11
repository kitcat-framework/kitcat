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

type EventHandlerResultStatus string

const (
	EventHandlerResultStatusFailed   EventHandlerResultStatus = "FAILED"
	EventHandlerResultStatusSuccess  EventHandlerResultStatus = "SUCCESS"
	EventHandlerResultStatusPending  EventHandlerResultStatus = "PENDING"
	EventHandlerResultStatusTakeable EventHandlerResultStatus = "TAKEABLE"
)

type HandlerResult struct {
	ID          int32
	HandlerName string

	EventID int32
	Event   *Event

	Status EventHandlerResultStatus `gorm:"index"`
	Error  *string

	RetryNumber     int32
	MaxRetries      int32
	RetryIntervalMs int64

	CreatedAt     pgtype.Timestamp `gorm:"type:timestamp"`
	UpdatedAt     pgtype.Timestamp `gorm:"type:timestamp"`
	ProcessableAt pgtype.Timestamp `gorm:"type:timestamp"`

	RunAt             *pgtype.Timestamp `gorm:"type:timestamp"`
	HandlerDurationMs int64

	FailedAt  *pgtype.Timestamp `gorm:"type:timestamp"`
	SuccessAt *pgtype.Timestamp `gorm:"type:timestamp"`
	PendingAt *pgtype.Timestamp `gorm:"type:timestamp"`
}

func (HandlerResult) TableName() string {
	return "kitevent.handler_results"
}
