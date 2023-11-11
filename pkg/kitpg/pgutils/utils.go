package pgutils

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

func TimestampUTC(time time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: time.UTC(), Valid: true}
}

func Timestamp(time time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: time, Valid: true}
}
