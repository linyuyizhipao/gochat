/**
 * Created by lock
 * Date: 2019-09-22
 * Time: 22:37
 */
package db

import (
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

const (
	DefaultDbname   = "gochat"
	MysqlDatasource = "root:123456@tcp(127.0.0.1:3306)/gochat?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
)

var dbMap = map[string]*gorm.DB{}
var syncLock sync.Mutex

func init() {
	initDB(DefaultDbname)
}

//root:123456@tcp(127.0.0.1:3306)/platform?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
func initDB(dbName string) {
	var e error
	// if prod env , you should change mysql driver for yourself !!!
	syncLock.Lock()
	dbMap[dbName], e = gorm.Open(mysql.Open(MysqlDatasource))
	if config.GetMode() == "dev" {
		dbMap[dbName].Debug()
	}
	syncLock.Unlock()
	if e != nil {
		logrus.Error("connect db fail:%s", e.Error())
	}
}

func GetDb(dbName string) (db *gorm.DB) {
	if db, ok := dbMap[dbName]; ok {
		return db
	} else {
		return nil
	}
}

type DbGoChat struct {
}

func (*DbGoChat) GetDbName() string {
	return "gochat"
}
