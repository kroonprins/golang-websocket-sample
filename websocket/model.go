package websocket

import (
	"github.com/gorilla/websocket"
)

type WsMessage struct {
	Type int
	Body WsMessageBody
}

type WsMessageBody struct {
	Type    string
	Content interface{}
}

type WsConnection struct {
	conn                *websocket.Conn
	requests            chan WsMessage
	responses           chan WsMessage
	requestDeserializer func([]byte) WsMessageBody
	responseSerializer  func(WsMessageBody) []byte
	messageHandlers     map[string]func(WsMessage) WsMessage
}
