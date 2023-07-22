package mq

import (
	"context"
	"encoding/json"
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
	"strings"
	"sync"
	"time"
)

func newSyncData() *syncData {
	return &syncData{
		LastTime: time.Second * 10,
		offset:   0,
		count:    100,
		db:       db.GetDb(db.DefaultDbname),
		rds:      redisclient.Rds,
		once:     sync.Once{},
	}
}

type syncData struct {
	LastTime time.Duration //只保留多久之内的聊天记录
	offset   int64         //开始的offset
	count    int64         //每页数量
	db       *gorm.DB
	rds      *redis.Client
	once     sync.Once
}

func (s *syncData) pushRoom(ctx context.Context, redisMsgByte []byte) (err error) {
	redisChannel := config.QueueName
	if err = redisclient.Rds.LPush(redisChannel, redisMsgByte).Err(); err != nil {
		logrus.Errorf("logic,lpush chan err:%s", err.Error())
		return
	}
	return
}

// 一个小时执行一次
// 每次把所有的需要持久化到db的key全部扫一遍(基于redis)
// 目前只有2中类型的需要持久化的key，一类是房间聊天，一类是单聊
func (s *syncData) syncLoop() {

	for {
		s.sync()
		time.Sleep(s.LastTime)
	}

}

func (s *syncData) sync() {
	ctx := context.Background()
	offset := s.offset
	count := s.count

	for {

		opt := redis.ZRangeBy{
			Min:    "0",
			Max:    cast.ToString(time.Now().Add(-s.LastTime).Unix()),
			Offset: offset,
			Count:  count,
		}
		offset += count
		zs := s.rds.ZRangeByScoreWithScores(config.RedisPersistenceKeys, opt).Val()
		persistenceKeys := []string{}

		if len(zs) <= 0 {
			break
		}

		for _, z := range zs {
			persistenceKeys = append(persistenceKeys, cast.ToString(z.Member))
		}

		for _, persistenceKey := range persistenceKeys {
			if strings.Contains(persistenceKey, "room") {
				s.syncPersistencePushRoomKey(ctx, persistenceKey)
			} else {
				s.syncPersistencePushKey(ctx, persistenceKey)
			}

			if n := s.rds.ZCard(persistenceKey).Val(); n <= 0 {
				s.rds.ZRem(config.RedisPersistenceKeys, persistenceKey)
				continue
			}

			ss := time.Now().Add(s.LastTime).Unix() //每同步一次，下次再同步就是3天后了，这里有个问题，如果三天内触发了最大值，也需要立即flush db才行，这个后面再实现吧
			s.rds.ZAdd(config.RedisPersistenceKeys, redis.Z{
				Score:  float64(ss),
				Member: persistenceKey,
			})

		}

	}

}

func (s *syncData) syncPersistencePushRoomKey(ctx context.Context, persistenceKey string) {
	offset := s.offset
	count := s.count
	for {
		startTime := time.Now().Format("2006-01-02 15:04:05")
		persistenceOpt := redis.ZRangeBy{
			Min:    "0",
			Max:    "+inf",
			Offset: offset,
			Count:  count,
		}
		zs := s.rds.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		roomMessages, members := s.getRoomMessages(zs)
		if len(roomMessages) <= 0 {
			logrus.Infof("PersistencePush0 room  roomMessages=%+v zs:%+v ", roomMessages, zs)
			return
		}
		startTime1 := time.Now().Format("2006-01-02 15:04:05")
		offset += count

		//批量落库
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.CreateInBatches(roomMessages, len(roomMessages)).Error; err != nil {
				return err
			}
			return s.rds.ZRem(persistenceKey, members...).Err()
		})

		startTime2 := time.Now().Format("2006-01-02 15:04:05")
		if err != nil {
			logrus.Errorf("PersistencePush room  CreateInBatches err:%v ", err)
			return
		}
		logrus.Infof("PersistencePushInfo  roomMessages=%+v zs:%v startTime:%s startTime1=%s startTime2=%s", roomMessages, len(zs), startTime, startTime1, startTime2)
	}

}

func (s *syncData) syncPersistencePushKey(ctx context.Context, persistenceKey string) (isEmpty bool) {
	offset := s.offset
	count := s.count
	for {

		persistenceOpt := redis.ZRangeBy{
			Min:    "0",
			Max:    "+inf",
			Offset: offset,
			Count:  count,
		}
		offset += count
		zs := s.rds.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		minUid, maxUid := s.getMinUidMaxUid(zs)
		if minUid <= 0 || maxUid <= 0 {
			//正序拿的，如果minUid没拿到，说明没有需要持久化的
			logrus.Errorf("PersistencePush minUid maxUid is empty persistenceKey=%s；minUid=%d,maxUid=%d", persistenceKey, minUid, maxUid)
			return
		}

		userMsgSession := &dao.UserMsgSession{}
		s.db.WithContext(ctx).Where("min_uid", minUid).Where("max_uid", maxUid).First(userMsgSession)
		sessionId := userMsgSession.ID
		userMessages, members := s.getPushMessages(sessionId, zs)
		if len(userMessages) <= 0 {
			return
		}

		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if sessionId <= 0 {
				userMsgSession.MinUid = minUid
				userMsgSession.MaxUid = maxUid
				userMsgSession.CreateTime = time.Now()
				tx.Create(userMsgSession)
				sessionId = userMsgSession.ID
				logrus.Infof("PersistencePush sessionId=%d", sessionId)
			}

			//重新覆盖赋值sessionid
			for _, message := range userMessages {
				message.SessionID = sessionId
			}

			if err := tx.CreateInBatches(userMessages, len(userMessages)).Error; err != nil {
				return err
			}

			return s.rds.ZRem(persistenceKey, members...).Err()
		})
		if err != nil {
			logrus.Infof("syncPersistencePushKey RedisClient.ZRem err:%v ", err)
			return
		}

	}

}

func (s *syncData) getMinUidMaxUid(zs []redis.Z) (minUid, maxUid int64) {
	if len(zs) <= 0 {
		return
	}
	memberStr := cast.ToString(zs[0].Member)
	sendMsg := &proto.Send{}
	if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
		logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
	}
	if tools.ParseNowDateTime(sendMsg.CreateTime) > time.Now().Add(-s.LastTime).Unix() {
		return
	}

	minUid, maxUid = cast.ToInt64(sendMsg.FromUserId), cast.ToInt64(sendMsg.ToUserId)
	if minUid > maxUid {
		minUid, maxUid = maxUid, minUid
	}
	logrus.Infof("PersistencePush sendMsg=%+v", sendMsg)
	return
}

func (s *syncData) getRoomMessages(zs []redis.Z) (roomMessages []*dao.RoomMessage, members []interface{}) {
	for _, z := range zs {
		seqId := cast.ToInt64(z.Score)
		memberStr := cast.ToString(z.Member)

		sendMsg := &proto.PersistenceData{}
		if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
			logrus.Infof("PersistencePush getRoomMessages json.Unmarshal err:%v;memberStr=%s ", err, memberStr)
		}
		tt := tools.ParseNowDateTime(sendMsg.CreateTime)
		if tt > time.Now().Add(-s.LastTime).Unix() {
			continue
		}
		sendMsg.SeqId = seqId

		roomMessage := &dao.RoomMessage{
			Rid:        cast.ToInt64(sendMsg.RoomId),
			SeqID:      sendMsg.SeqId,
			UID:        cast.ToInt64(sendMsg.FromUserId),
			Content:    sendMsg.Msg,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		members = append(members, z.Member)
		roomMessages = append(roomMessages, roomMessage)
	}
	return
}

func (s *syncData) getPushMessages(sessionId int64, zs []redis.Z) (userMessages []*dao.UserMessage, members []interface{}) {
	for _, z := range zs {
		memberStr := cast.ToString(z.Member)

		sendMsg := &proto.PersistenceData{}
		if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
			logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
		}

		if tools.ParseNowDateTime(sendMsg.CreateTime) > time.Now().Add(-s.LastTime).Unix() {
			continue
		}
		members = append(members, z.Member)
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

	return
}
