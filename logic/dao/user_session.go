package dao

import "time"

type UserMsgSession struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	MinUid     int64     `gorm:"column:uid;not null" json:"min_uid"`
	MaxUid     int64     `gorm:"column:to_uid;not null" json:"max_uid"`
	CreateTime time.Time `gorm:"column:create_time;primaryKey;default:CURRENT_TIMESTAMP" json:"create_time"`
}

func (u *UserMsgSession) TableName() string {
	return "user_chat_session"
}
