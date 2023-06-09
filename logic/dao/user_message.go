package dao

import "time"

type UserMessage struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	SessionID  int64     `gorm:"column:session_id;not null" json:"session_id"`
	SeqID      int64     `gorm:"column:seq_id;not null" json:"seq_id"`
	UID        int64     `gorm:"column:uid;not null" json:"uid"`
	ToUID      int64     `gorm:"column:to_uid;not null" json:"to_uid"`
	Content    string    `gorm:"column:content;not null" json:"content"`
	CreateTime time.Time `gorm:"column:create_time;primaryKey;default:CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP" json:"update_time"`
}

func (u *UserMessage) TableName() string {
	return "user_message"
}
