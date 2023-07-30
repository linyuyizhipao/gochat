package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/judwhite/go-svc"
	"github.com/spf13/cast"
	"gochat/proto"
	"sync"
	"sync/atomic"
	"time"
)

// program implements svc.Service
type program struct {
	wg   sync.WaitGroup
	quit chan struct{}

	tokens []string
}

func main() {
	prg := &program{}

	// Call svc.Run to start your program/service.
	if err := svc.Run(prg); err != nil {
		fmt.Println(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	for i := 0; i < 1; i++ {
		name := fmt.Sprintf("test-%d", i+1)
		authToken := login(name)
		if authToken == "" {
			fmt.Printf("authToken is empty %d", i)
			continue
		}
		p.tokens = append(p.tokens, authToken)
	}
	return nil
}

func (p *program) Start() error {
	// The Start method must not block, or Windows may assume your service failed
	// to start. Launch a Goroutine here to do something interesting/blocking.

	p.quit = make(chan struct{})
	for i, token := range p.tokens {
		token := token
		roomId := 1
		name := fmt.Sprintf("test-%d", i+1)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			websocket1(name, token, roomId)
		}()
	}

	return nil
}

func (p *program) Stop() error {
	// The Stop method is invoked by stopping the Windows service, or by pressing Ctrl+C on the console.
	// This method may block, but it's a good idea to finish quickly or your process may be killed by
	// Windows during a shutdown/reboot. As a general rule you shouldn't rely on graceful shutdown.

	fmt.Println("Stopping...")
	close(p.quit)
	p.wg.Wait()
	fmt.Println("Stopped.")
	return nil
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

var nnm int64
var gg sync.Once

func websocket1(name, authToken string, roomId int) {
	// 连接WebSocket服务器
	conn, _, err := websocket.DefaultDialer.Dial("ws://api.gochat.com:7000/ws", nil)
	if err != nil {
		fmt.Println("err111111111:", err)
		return
	}
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
	quit := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer func() {
			ticker.Stop()
			conn.Close()
		}()
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	go func() {
		gg.Do(func() {
			for {
				fmt.Printf("nnmnnmnnmnnmnnm=%d \n", atomic.LoadInt64(&nnm))
				time.Sleep(time.Second * 3)
			}
		})

	}()

	uid, userName := getUser(authToken)
	for j := 0; j < 100; j++ {
		atomic.AddInt64(&nnm, 1)
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
	quit <- struct{}{}
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
