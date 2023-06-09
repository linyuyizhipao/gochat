/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 18:25
 */
package logic

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"runtime"
)

type Logic struct {
	ServerId string
}

func New() *Logic {
	return new(Logic)
}

func (logic *Logic) Run() {
	//read config
	runtime.GOMAXPROCS(runtime.NumCPU())
	logic.ServerId = fmt.Sprintf("logic-%s", uuid.New().String())
	//init publish redis
	if err := logic.InitPublishRedisClient(); err != nil {
		logrus.Panicf("logic init publishRedisClient fail,err:%s", err.Error())
	}

	//init rpc server
	if err := logic.InitRpcServer(); err != nil {
		logrus.Panicf("logic init rpc server fail")
	}
}
