package connect

import (
	"context"
	"github.com/sirupsen/logrus"
	"gochat/proto"
)

type RpcConnect struct {
}

func (rpc *RpcConnect) Connect(connReq *proto.ConnectRequest) (uid int, err error) {
	reply := &proto.ConnectReply{}
	err = logicRpcClient.Call(context.Background(), "Connect", connReq, reply)
	if err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	uid = reply.UserId
	logrus.Infof("connect logic userId :%d", reply.UserId)
	return
}

func (rpc *RpcConnect) DisConnect(disConnReq *proto.DisConnectRequest) (err error) {
	reply := &proto.DisConnectReply{}
	if err = logicRpcClient.Call(context.Background(), "DisConnect", disConnReq, reply); err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	return
}

func (rpc *RpcConnect) Push(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	logicRpcClient.Call(context.Background(), "Push", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

// 调用loginc的rpc，进行房间消息推送.logic会将消息发送到消息队列，给task消费进行最后的推送
func (rpc *RpcConnect) PushRoom(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	logicRpcClient.Call(context.Background(), "PushRoom", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}
