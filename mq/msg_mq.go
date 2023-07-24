package mq

import (
	"context"
	"gochat/mq/internal"
)

type (
	Queue interface {
		Enqueue(ctx context.Context, msg []byte) (err error)
		Dequeue(ctx context.Context) (msg []byte, err error)
	}
	Persistence interface {
		Write(ctx context.Context, msgId int64, body []byte) (msg []byte, err error)
		FlushWhiteLoop(ctx context.Context) (err error)
	}
)

func GetQueue() Queue {
	return internal.NewRdsQueue()
}

func GetPersistence() Persistence {
	return internal.NewRdsPersistence()
}

type Mq struct {
	q Queue
}

func NewMq() *Mq {
	return &Mq{
		q: GetQueue(),
	}
}

func (k *Mq) SendMsg(ctx context.Context, redisMsgByte []byte) (err error) {
	return k.q.Enqueue(ctx, redisMsgByte)
}

func (k *Mq) ConsumeMsg(ctx context.Context) (msg []byte, err error) {
	return k.q.Dequeue(ctx)
}
