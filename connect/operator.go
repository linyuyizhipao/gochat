/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:35
 */
package connect

import "gochat/proto"

type Operator interface {
	Connect(conn *proto.ConnectRequest) (int, error)
	DisConnect(disConn *proto.DisConnectRequest) (err error)
	Push(req *proto.Send) (code int, msg string)
	PushRoom(req *proto.Send) (code int, msg string)
}

type DefaultOperator struct {
}

// rpc call logic layer
func (o *DefaultOperator) Connect(conn *proto.ConnectRequest) (uid int, err error) {
	rpcConnect := new(RpcConnect)
	uid, err = rpcConnect.Connect(conn)
	return
}

// rpc call logic layer
func (o *DefaultOperator) DisConnect(disConn *proto.DisConnectRequest) (err error) {
	rpcConnect := new(RpcConnect)
	err = rpcConnect.DisConnect(disConn)
	return
}

func (o *DefaultOperator) Push(sendMsg *proto.Send) (code int, msg string) {
	rpcConnect := new(RpcConnect)
	code, msg = rpcConnect.Push(sendMsg)
	return
}

// rpc call logic layer
func (o *DefaultOperator) PushRoom(sendMsg *proto.Send) (code int, msg string) {
	rpcConnect := new(RpcConnect)
	code, msg = rpcConnect.PushRoom(sendMsg)
	return
}
