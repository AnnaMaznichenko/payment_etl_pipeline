package dbo

import "time"

type ETLCheckpoint struct {
	SourceName      string     `gorm:"primaryKey;column:source_name"`
	LastProcessedAt *time.Time `gorm:"column:last_processed_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (ETLCheckpoint) TableName() string {
	return "etl_checkpoints"
}
