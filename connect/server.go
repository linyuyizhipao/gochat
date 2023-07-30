/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:32
 */
package connect

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/proto"
	"gochat/tools"
	"time"
)

type Server struct {
	Buckets   []*Bucket
	Options   ServerOptions
	bucketIdx uint32
	operator  Operator
}

type ServerOptions struct {
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
	BroadcastSize   int
}

func NewServer(b []*Bucket, o Operator, options ServerOptions) *Server {
	s := new(Server)
	s.Buckets = b
	s.Options = options
	s.bucketIdx = uint32(len(b))
	s.operator = o
	return s
}

// reduce lock competition, use google city hash insert to different bucket
func (s *Server) Bucket(userId int) *Bucket {
	userIdStr := fmt.Sprintf("%d", userId)
	idx := tools.CityHash32([]byte(userIdStr), uint32(len(userIdStr))) % s.bucketIdx
	return s.Buckets[idx]
}

func (s *Server) writePump(ch *Channel, c *Connect) {
	//PingPeriod default eq 54s
	ticker := time.NewTicker(s.Options.PingPeriod)
	defer func() {
		ticker.Stop()
		ch.conn.Close()
	}()

	for {
		select {
		case message, ok := <-ch.broadcast:
			//write data dead time , like http timeout , default 10s
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			if !ok {
				logrus.Warn("SetWriteDeadline not ok")
				ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := ch.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logrus.Warn(" ch.conn.NextWriter err :%s  ", err.Error())
				return
			}
			logrus.Infof("message write body:%s", message.Body)
			w.Write(message.Body)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			//heartbeat，if ping error will exit and close current websocket conn
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			logrus.Infof("websocket.PingMessage :%v", websocket.PingMessage)
			if err := ch.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) readPump(ch *Channel, c *Connect) {
	defer func() {
		if ee := recover(); ee != nil {
			logrus.Errorf("start exec disConnect ...%v", ee)
		}

		logrus.Infof("start exec disConnect ...")
		if ch.Room == nil || ch.userId == 0 {
			logrus.Infof("roomId and userId eq 0")
			ch.conn.Close()
			return
		}
		logrus.Infof("exec disConnect ...")
		disConnectRequest := new(proto.DisConnectRequest)
		disConnectRequest.RoomId = ch.Room.Id
		disConnectRequest.UserId = ch.userId
		s.Bucket(ch.userId).DeleteChannel(ch)
		if err := s.operator.DisConnect(disConnectRequest); err != nil {
			logrus.Warnf("DisConnect err :%s", err.Error())
		}
		ch.conn.Close()
	}()

	ch.conn.SetReadLimit(s.Options.MaxMessageSize)
	ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
	ch.conn.SetPongHandler(func(string) error {
		ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
		return nil
	})

	for {
		_, message, err := ch.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("readPump1 ReadMessage err:%s", err.Error())
				return
			}
		}
		if message == nil {
			logrus.Errorf("readPump2 ReadMessage message:%s", message)
			continue
		}

		var sendMsg proto.SendWebSocket
		if err := json.Unmarshal(message, &sendMsg); err != nil {
			logrus.Errorf("readPump3 tcp message struct %+v", sendMsg)
			break
		}
		if err = s.Exec(c.ServerId, sendMsg, ch); err != nil {
			logrus.Errorf("tcp message Exec %v", err)
			return
		}
	}
}

func (s *Server) Exec(serverId string, sendMsg proto.SendWebSocket, ch *Channel) (err error) {
	switch sendMsg.Op {
	case config.OpWebsocketConnSub:
		return s.Sub(serverId, sendMsg, ch)
	case config.OpSingleSend:
		return s.Push(sendMsg)
	case config.OpRoomSend:
		return s.PushRoom(sendMsg)
	default:
		return s.Sub(serverId, sendMsg, ch)
	}
}

func (s *Server) Sub(serverId string, sendMsg proto.SendWebSocket, ch *Channel) (err error) {
	if sendMsg.AuthToken == "" {
		err = errors.New("tcp s.operator.Connect no authToken")
		return
	}
	var connReq = &proto.ConnectRequest{}
	var userId int
	connReq.RoomId = sendMsg.RoomId
	connReq.AuthToken = sendMsg.AuthToken
	connReq.ServerId = serverId //config.Conf.Connect.ConnectWebsocket.ServerId
	userId, err = s.operator.Connect(connReq)
	if err != nil {
		err = errors.Errorf("s.operator.Connect error %s", err.Error())
		return
	}
	if userId == 0 {
		err = errors.New("Invalid AuthToken ,userId empty")
		return
	}
	logrus.Infof("websocket rpc call return userId:%d,RoomId:%d", userId, connReq.RoomId)
	b := s.Bucket(userId)
	err = b.Put(userId, connReq.RoomId, ch)
	if err != nil {
		cErr := ch.conn.Close()
		logrus.Errorf("conn close err: %s; cErr=%v", err.Error(), cErr)
		return
	}
	return nil
}

func (s *Server) Push(sendMsg proto.SendWebSocket) (err error) {
	req := &proto.Send{
		Msg:          sendMsg.Msg,
		FromUserId:   sendMsg.FromUserId,
		FromUserName: sendMsg.FromUserName,
		ToUserId:     sendMsg.ToUserId,
		ToUserName:   sendMsg.ToUserName,
		RoomId:       sendMsg.RoomId,
		Op:           config.OpSingleSend,
	}

	code, rpcMsg := s.operator.PushMsg(req)
	if code == tools.CodeFail {
		logrus.Errorf("Server Push code: %v; rpcMsg=%v", code, rpcMsg)
		return
	}
	return nil
}

func (s *Server) PushRoom(sendMsg proto.SendWebSocket) (err error) {
	req := &proto.Send{
		Msg:          sendMsg.Msg,
		FromUserId:   sendMsg.FromUserId,
		FromUserName: sendMsg.FromUserName,
		RoomId:       sendMsg.RoomId,
		Op:           config.OpRoomSend,
	}

	code, rpcMsg := s.operator.PushRoom(req)
	if code == tools.CodeFail {
		logrus.Errorf("Server PushRoom code: %v; rpcMsg=%v", code, rpcMsg)
		return
	}
	return nil
}
