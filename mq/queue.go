package mq

import (
	"context"
)

type MsgDecomposer interface {
	SendMsg(ctx context.Context, msg []byte) (err error)
	ConsumeMsg(ctx context.Context) (msg []byte, err error)

	InitPersistence()
	DataPersistencePush(ctx context.Context, uid int, msgId int64, messages []byte) (err error)
	DataPersistencePushRoom(ctx context.Context, roomId int, msgId int64, body []byte) (err error)
}

func GetMsgDecomposer() MsgDecomposer {
	return NewRedisMq()
}
