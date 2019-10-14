package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Upgrader struct {
	websocket.Upgrader
}

func HandleFunc(pattern string, handler func(*WsConnection), upgrader Upgrader) {
	HandleFunc := func(w http.ResponseWriter, r *http.Request) {
		wsConnection, err := wsConnection(w, r, upgrader.Upgrader)
		if err != nil {
			log.Print("ws connection setup failed:", err)
			return
		}
		defer wsConnection.Close()

		handler(wsConnection)

		log.Print("Connection closed")
	}
	http.HandleFunc(pattern, HandleFunc)
}

func wsConnection(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) (*WsConnection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return nil, err
	}

	requests := make(chan WsMessage)
	responses := make(chan WsMessage)

	wsConnection := WsConnection{conn: conn, requests: requests, responses: responses, messageHandlers: make(map[string]func(WsMessage) WsMessage)}
	return &wsConnection, nil
}

func (wsConnection *WsConnection) Listen() {
	go wsConnection.handleRequests()
	go wsConnection.handleResponses()
	wsConnection.processRequests()
}

func (wsConnection *WsConnection) Close() error {
	return wsConnection.conn.Close()
}

func (wsConnection *WsConnection) MessageHandleFunc(messageType string, handleFunc func(WsMessage) WsMessage) {
	wsConnection.messageHandlers[messageType] = handleFunc
}

func (wsConnection *WsConnection) RequestDeserializer(requestDeserializer func([]byte) WsMessageBody) {
	wsConnection.requestDeserializer = requestDeserializer
}

func (wsConnection *WsConnection) ResponseSerializer(responseSerializer func(WsMessageBody) []byte) {
	wsConnection.responseSerializer = responseSerializer
}

func (wsConnection *WsConnection) handleResponses() {
	for response := range wsConnection.responses {
		wsConnection.conn.WriteMessage(response.Type, wsConnection.responseSerializer(response.Body))
	}
}

func (wsConnection *WsConnection) handleRequests() {
	for {
		messageType, message, err := wsConnection.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			close(wsConnection.requests)
			close(wsConnection.responses)
			break
		} else {
			wsConnection.requests <- WsMessage{Type: messageType, Body: wsConnection.requestDeserializer(message)}
		}
	}
}

func (wsConnection *WsConnection) processRequests() {
	for request := range wsConnection.requests {
		log.Printf("recv: %s", request.Body)
		go func(request WsMessage) {
			message := wsConnection.messageHandlers[request.Body.Type](request)
			select {
			case <-wsConnection.responses:
				log.Println("not sending because connection was closed in the meantime")
				return
			default:
				wsConnection.responses <- message
			}
		}(request)
	}
}
