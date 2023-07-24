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
	md := mq.NewMq()
	persistence := mq.GetPersistence()

	go func() {
		_ = persistence.FlushWhiteLoop(context.Background())
	}()

	go func() {
		logrus.Info("InitQueueClient len(result)")

		for {
			msg, err := md.ConsumeMsg(context.Background())
			if err != nil {
				logrus.Errorf("task queue block timeout,no msg err:%s", err.Error())
				return
			}
			if len(msg) <= 0 {
				logrus.Errorf("task queue block no msg err:%s", err.Error())
				continue
			}
			task.Push(string(msg))
		}
	}()
	return
}
