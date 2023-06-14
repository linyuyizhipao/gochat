/**
 * Created by lock
 * Date: 2019-08-18
 * Time: 18:03
 */
package tools

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

const SessionPrefix = "sess_"

var (
	nodeMap = map[int64]*snowflake.Node{}
)

func GetSnowflakeId(nodeIds ...int64) int64 {
	nodeId := int64(1) //default node id eq 1,this can modify to different serverId node
	if len(nodeIds) <= 0 {
		nodeId = nodeIds[0]
	}

	if node, ok := nodeMap[nodeId]; ok {
		return node.Generate().Int64()
	}
	node, err := snowflake.NewNode(nodeId)
	if err != nil {
		logrus.Errorf("GetSnowflakeId nodeId=%d;err=%v", nodeId, err)
		return 0
	}
	nodeMap[nodeId] = node
	return node.Generate().Int64()
}

func GetRandomToken(length int) string {
	r := make([]byte, length)
	io.ReadFull(rand.Reader, r)
	return base64.URLEncoding.EncodeToString(r)
}

func CreateSessionId(sessionId string) string {
	return SessionPrefix + sessionId
}

func GetSessionIdByUserId(userId int) string {
	return fmt.Sprintf("sess_map_%d", userId)
}

func GetSessionName(sessionId string) string {
	return SessionPrefix + sessionId
}

func Sha1(s string) (str string) {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func GetNowDateTime() string {
	return time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
}

func ParseNowDateTime(t string) int64 {
	nowT, err := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	if err != nil {
		return 0
	}
	return nowT.Unix()
}

func GenerateUserMsgKey(userId, toUserId int) string {
	if userId < toUserId {
		return fmt.Sprintf("%d_%d", userId, toUserId)
	}
	return fmt.Sprintf("%d_%d", toUserId, userId)
}
