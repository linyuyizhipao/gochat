package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cast"
	"strings"
	"sync"
)

func main() {
	wg := &sync.WaitGroup{}
	content := ""
	authToken := getToken()
	fmt.Println(authToken, "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg")
	for i := 0; i < 100; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				content = fmt.Sprintf("我是这是什么话 i:%d; j:%d", i, j)
				pushRoom(1, authToken, content)
			}
		}()
	}

	wg.Wait()

}

func getToken() string {
	type FormLogin struct {
		UserName string `form:"userName" json:"userName" binding:"required"`
		Password string `form:"passWord" json:"passWord" binding:"required"`
	}
	formRoom := FormLogin{
		UserName: "hugh",
		Password: "111111",
	}
	bytes, _ := json.Marshal(formRoom)

	url := "http://127.0.0.1:7070/user/login"
	body := bytes
	client := resty.New()
	resp, err := client.R().SetHeader("Accept", "application/json").SetBody(body).Post(url)
	if err != nil {
		fmt.Printf("错了:%s", err.Error())
		return ""
	}
	respMap := map[string]interface{}{}
	json.Unmarshal(resp.Body(), &respMap)
	respData := respMap["data"]
	respData2 := cast.ToString(respData)
	return respData2

}

func pushRoom(rid int, authToken string, content string) {
	type FormRoom struct {
		AuthToken string `form:"authToken" json:"authToken" binding:"required"`
		Msg       string `form:"msg" json:"msg" binding:"required"`
		RoomId    int    `form:"roomId" json:"roomId" binding:"required"`
	}
	formRoom := FormRoom{
		AuthToken: authToken,
		Msg:       content,
		RoomId:    rid,
	}
	bytes, _ := json.Marshal(formRoom)

	url := "http://127.0.0.1:7070/push/pushRoom"
	body := bytes
	client := resty.New()
	resp, err := client.R().SetHeader("Accept", "application/json").SetBody(body).Post(url)
	if err != nil {
		fmt.Printf("错了:%s", err.Error())
		return
	}

	if !strings.Contains(string(resp.Body()), "ok") {
		fmt.Println("错了222:" + string(resp.Body()))
		panic(1)
	}
}
