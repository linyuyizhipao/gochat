package connect

import (
	"gochat/proto"
)

type RpcConnect struct {
}

func (rpc *RpcConnect) Connect(connReq *proto.ConnectRequest) (uid int, err error) {
	return logicRpcClient.Connect(connReq)
}

func (rpc *RpcConnect) DisConnect(disConnReq *proto.DisConnectRequest) (err error) {
	return logicRpcClient.DisConnect(disConnReq)
}

func (rpc *RpcConnect) PushMsg(req *proto.Send) (code int, msg string) {
	return logicRpcClient.Push(req)
}

// 调用loginc的rpc，进行房间消息推送.logic会将消息发送到消息队列，给task消费进行最后的推送
func (rpc *RpcConnect) PushRoom(req *proto.Send) (code int, msg string) {
	return logicRpcClient.PushRoom(req)
}
