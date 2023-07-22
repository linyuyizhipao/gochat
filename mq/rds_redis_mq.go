package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/pkg/redisclient"
	"gochat/proto"
	"gochat/tools"
	"time"
)

type RedisMq struct {
}

func NewRedisMq() *RedisMq {
	return &RedisMq{}
}

func (k *RedisMq) InitPersistence() {
	go newSyncData().syncLoop()
}

func (k *RedisMq) SendMsg(ctx context.Context, redisMsgByte []byte) (err error) {
	syncDa := newSyncData()
	return syncDa.pushRoom(ctx, redisMsgByte)
}

func (k *RedisMq) ConsumeMsg(ctx context.Context) (msg []byte, err error) {
	result, rErr := redisclient.Rds.BRPop(time.Second*10, config.QueueName).Result()
	if rErr != nil && rErr != redis.Nil {
		err = rErr
		logrus.Errorf("task queue block timeout,no msg err:%s", err.Error())
		return
	}
	logrus.Infof("InitQueueClient len(result)=%d,result=%v", len(result), result)
	if len(result) >= 2 {
		msg = []byte(result[1])
	}
	return
}

func (k *RedisMq) DataPersistencePush(ctx context.Context, userId int, msgId int64, body []byte) (err error) {
	seqId := msgId
	sendMsg := &proto.Send{}
	persistenceData := &proto.PersistenceData{}
	if err := json.Unmarshal(body, sendMsg); err != nil {
		logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
	}
	persistenceData.Msg = sendMsg.Msg
	persistenceData.RoomId = sendMsg.RoomId
	persistenceData.FromUserId = sendMsg.FromUserId
	persistenceData.ToUserId = sendMsg.ToUserId
	persistenceData.Op = sendMsg.Op
	persistenceData.SeqId = seqId
	perDataBody, err := json.Marshal(persistenceData)
	if err != nil {
		logrus.Errorf("PersistencePush json.Marshal err:%v ", err)
		return
	}
	persistenceKey := fmt.Sprintf(config.RedisPushPersistence, tools.GenerateUserMsgKey(userId, sendMsg.ToUserId))
	d := redis.Z{
		Score:  float64(persistenceData.SeqId),
		Member: string(perDataBody),
	}
	pipe := redisclient.Rds.Pipeline()
	pipe.ZAdd(persistenceKey, d)
	pipe.ZAdd(config.RedisPersistenceKeys, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: persistenceKey,
	})
	_, err = pipe.Exec()
	if err != nil {
		logrus.Infof("PersistencePush Exec err:%v ", err)
		return
	}
	logrus.Infof("PersistencePush pushRoomMsgRequest,userId=%d,seqId=%d,body=%+v ", userId, seqId, body)
	return
}

func (k *RedisMq) DataPersistencePushRoom(ctx context.Context, roomId int, msgId int64, body []byte) (err error) {
	d := redis.Z{
		Score:  float64(msgId),
		Member: string(body),
	}
	persistenceKey := fmt.Sprintf(config.RedisPushRoomPersistence, roomId)
	pipe := redisclient.Rds.Pipeline()
	pipe.ZAdd(persistenceKey, d)
	pipe.ZAdd(config.RedisPersistenceKeys, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: persistenceKey,
	})
	_, err = pipe.Exec()
	if err != nil {
		logrus.Infof("PersistencePushRoom Exec err:%v ", err)
		return
	}
	return
}
