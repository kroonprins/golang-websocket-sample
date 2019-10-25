package websocket

import (
	"github.com/gorilla/websocket"
)

// WsMessage represents an incoming message on the websocket connection
type WsMessage struct {
	Type int
	Body WsMessageBody
}

// WsMessageBody represents the body of an incoming message on the websocket connection
type WsMessageBody struct {
	Type    string
	Content interface{}
}

type wsErrorMessage struct {
	Type  int
	Error error
}

// WsConnection represents the web socket connection
type WsConnection struct {
	conn                *websocket.Conn
	requests            chan WsMessage
	responses           chan WsMessage
	errors              chan wsErrorMessage
	requestDeserializer func([]byte) (*WsMessageBody, error)
	responseSerializer  func(WsMessageBody) ([]byte, error)
	messageHandlers     map[string]func(WsMessage) (*WsMessage, error)
	errorHandler        func(error) WsMessageBody
}
