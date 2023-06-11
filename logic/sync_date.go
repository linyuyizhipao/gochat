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
		time.Sleep(time.Hour * 24 * time.Duration(s.days))
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

			if n := RedisClient.ZCard(persistenceKey).Val(); n <= 0 {
				RedisClient.ZRem(config.RedisPersistenceKeys, persistenceKey)
				continue
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
		zs := RedisClient.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		roomMessages, members := s.getRoomMessages(zs)
		if len(roomMessages) <= 0 {
			return
		}

		//批量落库
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.CreateInBatches(roomMessages, len(roomMessages)).Error; err != nil {
				return err
			}
			return RedisClient.ZRem(persistenceKey, members...).Err()
		})
		if err != nil {
			logrus.Errorf("PersistencePush room  CreateInBatches err:%v ", err)
			return
		}
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
		zs := RedisClient.ZRangeByScoreWithScores(persistenceKey, persistenceOpt).Val()
		minUid, maxUid := s.getMinUidMaxUid(zs)
		if minUid <= 0 || maxUid <= 0 {
			//正序拿的，如果minUid没拿到，说明没有需要持久化的
			logrus.Errorf("PersistencePush minUid maxUid is empty persistenceKey=%s", persistenceKey)
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

			return RedisClient.ZRem(persistenceKey, members...).Err()
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
	if tools.ParseNowDateTime(sendMsg.CreateTime) > time.Now().Add(-time.Duration(s.days)*time.Hour*24).Unix() {
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
		memberStr := cast.ToString(z.Member)
		seqId := cast.ToInt64(z.Score)

		sendMsg := &proto.Send{}
		if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
			logrus.Infof("PersistencePush getRoomMessages json.Unmarshal err:%v ", err)
		}
		if tools.ParseNowDateTime(sendMsg.CreateTime) > time.Now().Add(-time.Duration(s.days)*time.Hour*24).Unix() {
			continue
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
		members = append(members, z.Member)
		roomMessages = append(roomMessages, roomMessage)
	}
	return
}

func (s *syncData) getPushMessages(sessionId int64, zs []redis.Z) (userMessages []*dao.UserMessage, members []interface{}) {
	for _, z := range zs {
		memberStr := cast.ToString(z.Member)
		seqId := cast.ToInt64(z.Score)

		sendMsg := &proto.Send{}
		if err := json.Unmarshal([]byte(memberStr), sendMsg); err != nil {
			logrus.Infof("PersistencePush json.Unmarshal err:%v ", err)
		}

		if tools.ParseNowDateTime(sendMsg.CreateTime) > time.Now().Add(-time.Duration(s.days)*time.Hour*24).Unix() {
			continue
		}
		members = append(members, z.Member)
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
	}

	return
}
