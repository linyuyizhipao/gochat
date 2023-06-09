package rpcclient

import (
	"context"
	"github.com/rpcxio/libkv/store"
	etcdV3 "github.com/rpcxio/rpcx-etcd/client"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
	"gochat/config"
	"gochat/proto"
	"sync"
	"time"
)

type LogicRpc struct {
	client client.XClient
}

var logicRpcClient *LogicRpc
var once sync.Once

func GetLogicRpcClient() (logicClient *LogicRpc) {
	etcdConfigOption := &store.Config{
		ClientTLS:         nil,
		TLS:               nil,
		ConnectionTimeout: time.Duration(config.Conf.Common.CommonEtcd.ConnectionTimeout) * time.Second,
		Bucket:            "",
		PersistConnection: true,
		Username:          config.Conf.Common.CommonEtcd.UserName,
		Password:          config.Conf.Common.CommonEtcd.Password,
	}
	once.Do(func() {
		d, e := etcdV3.NewEtcdV3Discovery(
			config.Conf.Common.CommonEtcd.BasePath,
			config.Conf.Common.CommonEtcd.ServerPathLogic,
			[]string{config.Conf.Common.CommonEtcd.Host},
			true,
			etcdConfigOption,
		)
		if e != nil {
			logrus.Fatalf("init connect rpc etcd discovery client fail:%s", e.Error())
		}
		logicRpcClient = &LogicRpc{
			client: client.NewXClient(config.Conf.Common.CommonEtcd.ServerPathLogic, client.Failtry, client.RandomSelect, d, client.DefaultOption),
		}
	})
	if logicRpcClient == nil {
		panic("GetLogicRpcClient logicRpcClient is nol")
	}
	return logicRpcClient
}

func (r *LogicRpc) PersistencePush(ctx context.Context, body *proto.PushMsgRequest) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "PersistencePush", body, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (r *LogicRpc) PersistencePushRoom(ctx context.Context, body *proto.PushRoomMsgRequest) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "PersistencePushRoom", body, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

//~~~~~~~~~~~~~~~~~~~~~~~~~

func (r *LogicRpc) Login(req *proto.LoginRequest) (code int, authToken string, msg string) {
	reply := &proto.LoginResponse{}
	err := r.client.Call(context.Background(), "Login", req, reply)
	if err != nil {
		msg = err.Error()
	}
	code = reply.Code
	authToken = reply.AuthToken
	return
}

func (r *LogicRpc) Register(req *proto.RegisterRequest) (code int, authToken string, msg string) {
	reply := &proto.RegisterReply{}
	err := r.client.Call(context.Background(), "Register", req, reply)
	if err != nil {
		msg = err.Error()
	}
	code = reply.Code
	authToken = reply.AuthToken
	return
}

func (r *LogicRpc) GetUserNameByUserId(req *proto.GetUserInfoRequest) (code int, userName string) {
	reply := &proto.GetUserInfoResponse{}
	r.client.Call(context.Background(), "GetUserInfoByUserId", req, reply)
	code = reply.Code
	userName = reply.UserName
	return
}

func (r *LogicRpc) CheckAuth(req *proto.CheckAuthRequest) (code int, userId int, userName string) {
	reply := &proto.CheckAuthResponse{}
	r.client.Call(context.Background(), "CheckAuth", req, reply)
	code = reply.Code
	userId = reply.UserId
	userName = reply.UserName
	return
}

func (r *LogicRpc) Logout(req *proto.LogoutRequest) (code int) {
	reply := &proto.LogoutResponse{}
	r.client.Call(context.Background(), "Logout", req, reply)
	code = reply.Code
	return
}

func (r *LogicRpc) Push(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "Push", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (r *LogicRpc) PushRoom(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "PushRoom", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (r *LogicRpc) Count(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "Count", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (r *LogicRpc) GetRoomInfo(req *proto.Send) (code int, msg string) {
	reply := &proto.SuccessReply{}
	r.client.Call(context.Background(), "GetRoomInfo", req, reply)
	code = reply.Code
	msg = reply.Msg
	return
}

func (r *LogicRpc) Connect(connReq *proto.ConnectRequest) (uid int, err error) {
	reply := &proto.ConnectReply{}
	err = r.client.Call(context.Background(), "Connect", connReq, reply)
	if err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	uid = reply.UserId
	logrus.Infof("connect logic userId :%d", reply.UserId)
	return
}

func (r *LogicRpc) DisConnect(disConnReq *proto.DisConnectRequest) (err error) {
	reply := &proto.DisConnectReply{}
	if err = r.client.Call(context.Background(), "DisConnect", disConnReq, reply); err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	return
}
