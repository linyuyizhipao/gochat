package logic

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gochat/config"
	"gochat/db"
	"gochat/logic/dao"
	"gochat/proto"
	"gochat/tools"
	"strings"
	"time"
)

type syncData struct {
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
	days := 3
	offset := int64(0)
	count := int64(30)

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
	db := db.GetDb(db.DefaultDbname)
	days := 3
	offset := int64(0)
	count := int64(30)
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

		db.WithContext(ctx).CreateInBatches(roomMessages, len(roomMessages))

	}

}

func (s *syncData) syncPersistencePushKey(ctx context.Context, persistenceKey string) {
	db := db.GetDb(db.DefaultDbname)
	days := 3
	offset := int64(0)
	count := int64(30)
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

		if len(zs) <= 0 {
			break
		}

		for _, z := range zs {
			memberStr := cast.ToString(z.Member)
			seqId := cast.ToInt64(z.Score)

			sendMsg := &proto.Send{}
			if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
				logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
			}

			userMessage := &dao.UserMessage{
				SessionID:  cast.ToInt64(sendMsg.RoomId),
				SeqID:      seqId,
				UID:        cast.ToInt64(sendMsg.FromUserId),
				ToUID:      cast.ToInt64(sendMsg.ToUserId),
				Content:    sendMsg.Msg,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			}
			userMessages = append(userMessages, userMessage)
		}

		db.WithContext(ctx).CreateInBatches(userMessages, len(userMessages))

	}

}
