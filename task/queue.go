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
	"time"
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
			msg, merr := md.ConsumeMsg(context.Background())
			if merr != nil {
				logrus.Errorf("task queue block timeout,no msg err:%s", merr.Error())
				time.Sleep(time.Second * 10)
				continue
			}
			if len(msg) <= 0 {
				logrus.Errorf("task queue block no msg")
				continue
			}
			task.Push(string(msg))
		}
	}()
	return
}
