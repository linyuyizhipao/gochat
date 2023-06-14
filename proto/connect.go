/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:00
 */
package proto

const (
	OpClient = iota
)

type Msg struct {
	Ver       int    `json:"ver"`  // protocol version
	Operation int    `json:"op"`   // operation for request
	SeqId     int64  `json:"seq"`  // sequence number chosen by client
	Body      []byte `json:"body"` // binary body bytes
}

type ConnSendMsg struct {
	Ver       int    `json:"ver"`        // protocol version
	Operation int    `json:"op"`         // operation for request
	SeqId     string `json:"seq"`        // sequence number chosen by client
	AuthToken string `json:"auth_token"` // sequence number chosen by client
	Body      []byte `json:"body"`       // binary body bytes
}

type PushMsgRequest struct {
	UserId int
	Msg    Msg
}

type PushRoomMsgRequest struct {
	RoomId int
	Msg    Msg
}

type PushRoomCountRequest struct {
	RoomId int
	Count  int
}
