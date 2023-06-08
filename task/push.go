/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:50
 */
package task

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/proto"
	"math/rand"
)

type PushParams struct {
	ServerId string
	UserId   int
	Msg      []byte
	RoomId   int
}

var pushChannel []chan *PushParams

func init() {
	pushChannel = make([]chan *PushParams, config.Conf.Task.TaskBase.PushChan)
}

func (task *Task) GoPush() {
	for i := 0; i < len(pushChannel); i++ {
		pushChannel[i] = make(chan *PushParams, config.Conf.Task.TaskBase.PushChanSize)
		go task.processSinglePush(pushChannel[i])
	}
}

// 单推，如果发生了用户漂移，那就广播
func (task *Task) processSinglePush(ch chan *PushParams) {
	var arg *PushParams
	for {
		arg = <-ch
		reply := task.pushSingleToConnect(arg.ServerId, arg.UserId, arg.Msg)
		if reply.Code == config.SuccessReplyCode {
			continue
		}

		if reply.Code == config.FailReplyCode && reply.Msg == config.FailReplyClientNotExistMsg {
			//如果下线又跑到别的serverid了，首先这个可以根据userid做负载均衡，去保证同一个userid会上同一个服务器
			//其次再发生的概率就相对更小了，实在发生就广播一下也无妨
			for serverId, _ := range RClient.ServerInsMap {
				if serverId == arg.ServerId {
					continue
				}

				reply := task.pushSingleToConnect(serverId, arg.UserId, arg.Msg)
				if reply.Code == config.SuccessReplyCode {
					break
				}

				if reply.Code == config.FailReplyCode && reply.Msg == config.FailReplyClientNotExistMsg {
					continue
				}
				break
			}
			continue
		}

	}
}

func (task *Task) Push(msg string) {
	m := &proto.RedisMsg{}
	if err := json.Unmarshal([]byte(msg), m); err != nil {
		logrus.Infof(" json.Unmarshal err:%v ", err)
	}
	logrus.Infof("push msg info %d,op is:%d", m.RoomId, m.Op)
	switch m.Op {
	case config.OpSingleSend:
		pushChannel[rand.Int()%config.Conf.Task.TaskBase.PushChan] <- &PushParams{
			ServerId: m.ServerId,
			UserId:   m.UserId,
			Msg:      m.Msg,
		}
	case config.OpRoomSend:
		task.broadcastRoomToConnect(m.RoomId, m.Msg)
	case config.OpRoomCountSend:
		task.broadcastRoomCountToConnect(m.RoomId, m.Count)
	case config.OpRoomInfoSend:
		task.broadcastRoomInfoToConnect(m.RoomId, m.RoomUserInfo)
	}
}
