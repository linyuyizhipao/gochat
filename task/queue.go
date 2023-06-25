/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:13
 */
package task

import (
	"context"
	"github.com/sirupsen/logrus"
	"gochat/mq"
)

func (task *Task) InitQueueClient() (err error) {
	go func() {
		logrus.Info("InitQueueRedisClient2 len(result222)")

		for {
			//10s timeout
			msg, err := mq.GetMsgDecomposer().ConsumeMsg(context.Background())
			if err != nil {
				logrus.Errorf("task queue block timeout,no msg err:%s", err.Error())
				return
			}
			if len(msg) > 0 {
				task.Push(string(msg))
			}
		}
	}()
	return
}
