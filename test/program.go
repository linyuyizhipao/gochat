package main

import (
	"fmt"
	"sync"
	"time"
)

// program implements svc.Service
type program struct {
	wg   sync.WaitGroup
	quit chan struct{}

	tokens []string
}

func main() {
	tokens := []string{}
	for i := 0; i < peopleNum; i++ {
		name := fmt.Sprintf("test-%d", i+1)
		authToken := login(name)
		if authToken == "" {
			fmt.Printf("authToken is empty %d", i)
			continue
		}
		tokens = append(tokens, authToken)
	}

	wg := sync.WaitGroup{}
	for i, token := range tokens {
		token := token
		roomId := 1
		name := fmt.Sprintf("test-%d", i+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			websocket1(name, token, roomId)
		}()
	}
	wg.Wait()
	fmt.Println("endendnednendendnednendendnedn")
	time.Sleep(time.Second * 10)

	for _, conn := range aaa {
		conn.Close()
	}
}
