package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
	"gochat/proto"
	"math/rand"
)

var wsUrls = []string{
	"ws://api.gochat.com:7000/ws", "ws://127.0.0.1:7000/ws",
	"ws://api.gochat.com:7001/ws", "ws://127.0.0.1:7001/ws",
	"ws://api.gochat.com:7002/ws", "ws://127.0.0.1:7002/ws",
	"ws://api.gochat.com:7003/ws", "ws://127.0.0.1:7003/ws",
	"ws://api.gochat.com:7004/ws", "ws://127.0.0.1:7004/ws",
	//"ws://api.gochat.com:7005/ws", "ws://127.0.0.1:7005/ws",
	//"ws://api.gochat.com:7006/ws", "ws://127.0.0.1:7006/ws",
}

var peopleNum = 500
var peopleMsgNum = 1

func register100() {
	num := peopleNum
	client := resty.New()
	request := client.R().ForceContentType("application/json")
	for i := 0; i < num; i++ {
		body := map[string]interface{}{
			"userName": fmt.Sprintf("test-%d", i+1),
			"passWord": "111111",
		}
		resp, err := request.SetBody(body).Post("http://127.0.0.1:7070/user/register")
		if err != nil {
			fmt.Printf("eeerr:%s", err.Error())
			return
		}
		dataByte := resp.Body()
		fmt.Printf("sss:%s", string(dataByte))
	}
}

func login(userName string) (authToken string) {
	client := resty.New()
	request := client.R().ForceContentType("application/json")
	body := map[string]interface{}{
		"userName": userName,
		"passWord": "111111",
	}
	resp, err := request.SetBody(body).Post("http://127.0.0.1:7070/user/login")
	if err != nil {
		fmt.Printf("loginerr:%s", err.Error())
		return
	}
	dataMap := map[string]interface{}{}
	_ = json.Unmarshal(resp.Body(), &dataMap)
	authToken = cast.ToString(dataMap["data"])
	return
}

type FormRoom struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
	Msg       string `form:"msg" json:"msg" binding:"required"`
	RoomId    int    `form:"roomId" json:"roomId" binding:"required"`
}

func pushRoom(authToken string, msg string, roomId int) {
	client := resty.New()
	request := client.R().ForceContentType("application/json")

	formRoom := FormRoom{
		AuthToken: authToken,
		Msg:       msg,
		RoomId:    roomId,
	}
	resp, err := request.SetBody(formRoom).Post("http://127.0.0.1:7070/push/pushRoom")
	if err != nil {
		fmt.Printf("respMsgerr:%s", err.Error())
		return
	}
	dataMap := map[string]interface{}{}
	_ = json.Unmarshal(resp.Body(), &dataMap)
	respMsg := cast.ToString(dataMap["msg"])
	if respMsg != "ok" {
		fmt.Printf("respMsg:%s", respMsg)
	}
	return
}

var aaa = map[string]*websocket.Conn{}

func websocket1(name, authToken string, roomId int) {
	// 连接WebSocket服务器
	wsUrl := wsUrls[rand.Intn(len(wsUrls))]
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		fmt.Println("err111111111:", err)
		return
	}
	aaa[authToken] = conn
	sendMsg := &proto.SendWebSocket{
		Code:         0,
		Msg:          "",
		FromUserId:   0,
		FromUserName: "",
		ToUserId:     0,
		ToUserName:   "",
		RoomId:       roomId,
		Op:           7,
		CreateTime:   "",
		AuthToken:    authToken,
	}
	sendMsgByte, _ := json.Marshal(sendMsg)

	// 发送消息
	err = conn.WriteMessage(websocket.TextMessage, sendMsgByte)
	if err != nil {
		fmt.Println("getUsergetUsergetUser 222err:", err)
		return
	}

	uid, userName := getUser(authToken)
	for j := 0; j < peopleMsgNum; j++ {
		sendMsg2 := &proto.SendWebSocket{
			Code:         0,
			Msg:          fmt.Sprintf("我是:%s,这是我说的第%d句话", name, j+1),
			FromUserId:   uid,
			FromUserName: userName,
			ToUserId:     0,
			ToUserName:   "",
			RoomId:       roomId,
			Op:           3,
			CreateTime:   "",
			AuthToken:    authToken,
		}
		sendMsgByte2, _ := json.Marshal(sendMsg2)
		err = conn.WriteMessage(websocket.TextMessage, sendMsgByte2)
		if err != nil {
			fmt.Printf("conn.WriteMessage err=%v", err)
			continue
		}
	}
	return
}

type FormCheckAuth struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
}

func getUser(authToken string) (uid int, userName string) {
	client := resty.New()
	request := client.R().ForceContentType("application/json")

	formRoom := FormCheckAuth{
		AuthToken: authToken,
	}
	resp, err := request.SetBody(formRoom).Post("http://127.0.0.1:7070/user/checkAuth")
	if err != nil {
		fmt.Printf("getUsererr:%s", err.Error())
		return
	}
	dataMap := map[string]interface{}{}
	err = json.Unmarshal(resp.Body(), &dataMap)
	if err != nil {
		fmt.Printf("getUsererr:%s", err.Error())
		return
	}
	data := cast.ToStringMap(dataMap["data"])
	uid = cast.ToInt(data["userId"])
	userName = cast.ToString(data["userName"])
	return
}
