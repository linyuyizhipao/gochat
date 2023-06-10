package logic

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gochat/config"
	"gochat/logic/dao"
	"gochat/proto"
	"gochat/tools"
	"gorm.io/gorm"
	"strings"
	"time"
)

type syncData struct {
	days   int   //只保留最近几天的聊天记录
	offset int64 //开始的offset
	count  int64 //每页数量
	db     *gorm.DB
}

// 一个小时执行一次
// 每次把所有的需要持久化到db的key全部扫一遍(基于redis)
// 目前只有2中类型的需要持久化的key，一类是房间聊天，一类是单聊
func (s *syncData) SyncLoop(ctx context.Context) {

	for {
		s.sync(ctx)
		time.Sleep(time.Minute)
	}

}

func (s *syncData) sync(ctx context.Context) {
	days := s.days
	offset := s.offset
	count := s.count

	for {

		opt := redis.ZRangeBy{
			Min:    "0",
			Max:    cast.ToString(time.Now().Add(-time.Duration(days) * time.Hour * 24).Unix()),
			Offset: offset,
			Count:  count,
		}
		offset += count
		zs := RedisClient.ZRangeByScoreWithScores(config.RedisPersistenceKeys, opt).Val()
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

			ss := time.Now().Add(time.Duration(days) * time.Hour * 24).Unix() //每同步一次，下次再同步就是3天后了，这里有个问题，如果三天内触发了最大值，也需要立即flush db才行，这个后面再实现吧
			RedisClient.ZAdd(config.RedisPersistenceKeys, redis.Z{
				Score:  float64(ss),
				Member: persistenceKey,
			})

		}

	}

}

func (s *syncData) syncPersistencePushRoomKey(ctx context.Context, persistenceKey string) {
	days := s.days
	offset := s.offset
	count := s.count
	minTime := time.Now().Add(-time.Duration(days) * time.Hour * 24).Unix()
	for {

		persistenceOpt := redis.ZRangeBy{
			Min:    "0",
			Max:    "+inf",
			Offset: offset,
			Count:  count,
		}
		offset += count
		zs := RedisClient.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		roomMessages := []*dao.RoomMessage{}

		if len(zs) <= 0 {
			RedisClient.ZRem(config.RedisPersistenceKeys, persistenceKey)
			break
		}

		for _, z := range zs {
			memberStr := cast.ToString(z.Member)
			seqId := cast.ToInt64(z.Score)

			sendMsg := &proto.Send{}
			if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
				logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
			}
			createTime := tools.ParseNowDateTime(sendMsg.CreateTime)
			if createTime > minTime {
				continue //后面可适配成return
			}
			roomMessage := &dao.RoomMessage{
				Rid:        cast.ToInt64(sendMsg.RoomId),
				SeqID:      seqId,
				UID:        cast.ToInt64(sendMsg.FromUserId),
				ToUID:      cast.ToInt64(sendMsg.ToUserId),
				Content:    sendMsg.Msg,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			}
			RedisClient.ZRem(persistenceKey, memberStr)
			roomMessages = append(roomMessages, roomMessage)
		}

		s.db.WithContext(ctx).CreateInBatches(roomMessages, len(roomMessages))

	}

}

func (s *syncData) syncPersistencePushKey(ctx context.Context, persistenceKey string) {
	days := s.days
	offset := s.offset
	count := s.count
	for {

		persistenceOpt := redis.ZRangeBy{
			Min:    "0",
			Max:    cast.ToString(time.Now().Add(-time.Duration(days) * time.Hour * 24)),
			Offset: offset,
			Count:  count,
		}
		offset += count
		zs := RedisClient.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		userMessages := []*dao.UserMessage{}
		userMsgSessions := []*dao.UserMsgSession{}

		minUid, maxUid := int64(0), int64(0)
		for _, z := range zs {
			memberStr := cast.ToString(z.Member)
			sendMsg := &proto.Send{}
			if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
				logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
			}
			minUid, maxUid = cast.ToInt64(sendMsg.FromUserId), cast.ToInt64(sendMsg.ToUserId)
			if minUid > maxUid {
				minUid, maxUid = maxUid, minUid
			}
		}
		if minUid <= 0 || maxUid <= 0 {
			logrus.Error("PersistencePush minUid maxUid is empty")
			continue
		}

		userMsgSession := &dao.UserMsgSession{
			MinUid:     minUid,
			MaxUid:     maxUid,
			CreateTime: time.Now(),
		}
		s.db.WithContext(ctx).Where("min_uid", minUid).Where("max_uid", maxUid).First(userMsgSession)
		sessionId := userMsgSession.ID
		if sessionId <= 0 {
			s.db.WithContext(ctx).Create(userMsgSession)
			sessionId = userMsgSession.ID
		}

		for _, z := range zs {
			memberStr := cast.ToString(z.Member)
			seqId := cast.ToInt64(z.Score)

			sendMsg := &proto.Send{}
			if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
				logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
			}

			userMessage := &dao.UserMessage{
				SessionID:  sessionId,
				SeqID:      seqId,
				UID:        cast.ToInt64(sendMsg.FromUserId),
				ToUID:      cast.ToInt64(sendMsg.ToUserId),
				Content:    sendMsg.Msg,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			}
			userMessages = append(userMessages, userMessage)
			userMsgSessions = append(userMsgSessions, userMsgSession)
		}

		s.db.WithContext(ctx).CreateInBatches(userMessages, len(userMessages))

	}

}
