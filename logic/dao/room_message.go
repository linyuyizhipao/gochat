package dao

import "time"

type RoomMessage struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Rid        int64     `gorm:"column:rid;not null" json:"rid"`
	SeqID      int64     `gorm:"column:seq_id;not null" json:"seq_id"`
	UID        int64     `gorm:"column:uid;not null" json:"uid"`
	Content    string    `gorm:"column:content;not null" json:"content"`
	CreateTime time.Time `gorm:"column:create_time;primaryKey;default:CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP" json:"update_time"`
}

func (u *RoomMessage) TableName() string {
	return "room_message"
}
