package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"

	"gochat/proto"
	"sync"
	"time"
)
import "github.com/go-resty/resty/v2"

// "2023-06-11 10:25:25"
func main() {
	wg := &sync.WaitGroup{}
	roomId := 1
	for i := 0; i < 10; i++ {
		wg.Add(1)
		num := i + 1
		name := fmt.Sprintf("test-%d", num)
		authToken := login(name)
		if authToken == "" {
			fmt.Printf("authToken is empty %d", i)
			return
		}
		go func(num int) {
			defer wg.Done()
			time.Sleep(time.Second * 5)
			websocket1(name, authToken, roomId)

			//for j := 0; j < 100; j++ {
			//	pushRoom(authToken, fmt.Sprintf("我是:%s,这是我说的第%d句话", name, j+1), roomId)
			//}
		}(num)

	}
	wg.Wait()

	time.Sleep(time.Second * 10)
	lock.Lock()

	for _, conn := range connMap {
		conn.Close()
	}
	lock.Unlock()

	fmt.Println("conn 结束了")
	time.Sleep(time.Hour)
}

func register100() {
	num := 100
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

var (
	connMap = map[string]*websocket.Conn{}
	lock    = &sync.Mutex{}
)

func websocket1(name, authToken string, roomId int) {
	// 连接WebSocket服务器
	conn, _, err := websocket.DefaultDialer.Dial("ws://api.gochat.com:7000/ws", nil)
	if err != nil {
		fmt.Println("err111111111:", err)
		return
	}
	lock.Lock()
	connMap[authToken] = conn
	lock.Unlock()
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
	for j := 0; j < 100; j++ {
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
	fmt.Printf("getUser222:%+v", dataMap)
	data := cast.ToStringMap(dataMap["data"])
	uid = cast.ToInt(data["userId"])
	userName = cast.ToString(data["userName"])
	return
}
