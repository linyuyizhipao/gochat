package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gochat/config"
	"gochat/db"
	"gochat/logic/dao"
	"gochat/pkg/redisclient"
	"gochat/proto"
	"gochat/tools"
	"gorm.io/gorm"
	"time"
)

type (
	RdsQueue struct {
		rds *redis.Client
	}
)

func NewRdsQueue() *RdsQueue {
	return &RdsQueue{
		rds: redisclient.Rds,
	}
}

func (rq *RdsQueue) Enqueue(ctx context.Context, msg []byte) (err error) {
	redisChannel := config.QueueName
	if err = rq.rds.LPush(redisChannel, msg).Err(); err != nil {
		logrus.Errorf("logic,lpush chan err:%s", err.Error())
		return
	}
	logrus.WithContext(ctx).Infof("Enqueue lmsg=%v", msg)
	return
}

func (rq *RdsQueue) Dequeue(ctx context.Context) (msg []byte, err error) {
	result, rErr := rq.rds.BRPop(0, config.QueueName).Result()
	if rErr != nil && rErr != redis.Nil {
		err = rErr
		logrus.Errorf("task queue block timeout,no msg err:%s", err.Error())
		return
	}
	logrus.WithContext(ctx).Infof("Dequeue len(result)=%d,result=%v", len(result), result)
	if len(result) >= 2 {
		msg = []byte(result[1])
		return
	}
	return
}

type (
	RdsPersistence struct {
		db         *gorm.DB
		flushQueue chan *proto.PersistenceData
	}
)

func NewRdsPersistence() *RdsPersistence {
	return &RdsPersistence{
		flushQueue: make(chan *proto.PersistenceData, 5000),
		db:         db.GetDb(db.DefaultDbname),
	}
}

// 定时批量刷盘
func (rq *RdsPersistence) FlushWhiteLoop(ctx context.Context) (err error) {
	msgs := make([]*proto.PersistenceData, 0, 50)
	for {
		isFlush := false
		select {
		case msg := <-rq.flushQueue:
			msgs = append(msgs, msg)
			isFlush = len(msgs) >= 50
		default:
			isFlush = true
		}
		if !isFlush {
			continue
		}

		//刷盘
		userMessages := []*dao.UserMessage{}
		roomMessages := []*dao.RoomMessage{}
		for _, sendMsg := range msgs {
			if sendMsg.RoomId > 0 {
				roomMessage := &dao.RoomMessage{
					Rid:        cast.ToInt64(sendMsg.RoomId),
					SeqID:      sendMsg.SeqId,
					UID:        cast.ToInt64(sendMsg.FromUserId),
					Content:    sendMsg.Msg,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}
				roomMessages = append(roomMessages, roomMessage)
			} else {
				minUid, maxUid := sendMsg.FromUserId, sendMsg.ToUserId
				if minUid > maxUid {
					minUid, maxUid = maxUid, minUid
				}
				userMsgSession := &dao.UserMsgSession{}
				rq.db.WithContext(ctx).Where("min_uid", minUid).Where("max_uid", maxUid).First(userMsgSession)
				sessionId := userMsgSession.ID
				if sessionId <= 0 {
					userMsgSession.MinUid = int64(minUid)
					userMsgSession.MaxUid = int64(maxUid)
					userMsgSession.CreateTime = time.Now()
					rq.db.Create(userMsgSession)
					sessionId = userMsgSession.ID
					logrus.Infof("PersistencePush sessionId=%d", sessionId)
				}
				userMessage := &dao.UserMessage{
					SessionID:  sessionId,
					SeqID:      sendMsg.SeqId,
					UID:        cast.ToInt64(sendMsg.FromUserId),
					ToUID:      cast.ToInt64(sendMsg.ToUserId),
					Content:    sendMsg.Msg,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}
				userMessages = append(userMessages, userMessage)
			}

		}
		if len(userMessages) > 0 {
			if err := rq.db.CreateInBatches(userMessages, len(userMessages)).Error; err != nil {
				return err
			}
		}

		if len(roomMessages) > 0 {
			if err := rq.db.CreateInBatches(roomMessages, len(roomMessages)).Error; err != nil {
				return err
			}
		}
		msgs = []*proto.PersistenceData{}
	}
}

func (rq *RdsPersistence) Write(ctx context.Context, msgId int64, body []byte) (msg []byte, err error) {
	seqId := msgId
	sendMsg := &proto.Send{}
	persistenceData := &proto.PersistenceData{}
	if err = json.Unmarshal(body, sendMsg); err != nil {
		logrus.WithContext(ctx).Infof("RdsPersistence json.Unmarshal err:%v ", err)
		return
	}
	persistenceData.Msg = sendMsg.Msg
	persistenceData.RoomId = sendMsg.RoomId
	persistenceData.FromUserId = sendMsg.FromUserId
	persistenceData.ToUserId = sendMsg.ToUserId
	persistenceData.Op = sendMsg.Op
	persistenceData.SeqId = seqId
	perDataBody, err := json.Marshal(persistenceData)
	if err != nil {
		logrus.Errorf("RdsPersistence json.Marshal err:%v ", err)
		return
	}
	persistenceKey := fmt.Sprintf(config.RedisPushRoomPersistence, sendMsg.RoomId, time.Now().Format(config.DayFmt))
	if sendMsg.RoomId <= 0 {
		persistenceKey = fmt.Sprintf(config.RedisPushPersistence, tools.GenerateUserMsgKey(sendMsg.FromUserId, sendMsg.ToUserId), time.Now().Format(config.DayFmt))
	}
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
	pipe.Expire(persistenceKey, config.DayExpired)
	_, err = pipe.Exec()
	if err != nil {
		logrus.Infof("RdsPersistence Exec err:%v ", err)
		return
	}
	rq.flushQueue <- persistenceData
	logrus.Infof("RdsPersistence pushRoomMsgRequest,userId=%d,seqId=%d,body=%+v ", sendMsg.FromUserId, seqId, body)
	return
}
