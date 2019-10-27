package websocket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader is a wrapper around gorilla github.com/gorilla/websocket/Upgrader
type Upgrader struct {
	websocket.Upgrader
}

// HandleFunc is used to add a websocket handler on the path specified by the pattern
func HandleFunc(pattern string, handler func(*WsConnection), upgrader Upgrader) {
	// TODO could be useful to add a handler for the upgrade step (e.g. handle session & authorization)
	// and a sesson context so that for every message we can now in which session context the message is happening
	HandleFunc := func(w http.ResponseWriter, r *http.Request) {
		wsConnection, err := wsConnection(w, r, upgrader.Upgrader)
		if err != nil {
			log.Print("ws connection setup failed:", err)
			return
		}
		defer wsConnection.close()

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
	errors := make(chan wsErrorMessage)

	wsConnection := WsConnection{
		conn:            conn,
		requests:        requests,
		responses:       responses,
		errors:          errors,
		messageHandlers: make(map[string]func(WsMessage) (*WsMessage, error)),
		errorHandler:    nil}
	return &wsConnection, nil
}

// Listen starts listening for websocket messages
func (wsConnection *WsConnection) Listen() {
	if wsConnection.requestDeserializer == nil {
		panic("A request deserializer must be registered on the websocket connection.")
	}
	if wsConnection.responseSerializer == nil {
		panic("A response serializer must be registered on the websocket connection.")
	}
	if len(wsConnection.messageHandlers) == 0 {
		panic("No message handlers have been registered for the websocket connection.")
	}

	go wsConnection.handleRequests()
	go wsConnection.handleResponses()
	go wsConnection.handleErrors()
	wsConnection.processRequests()
}

func (wsConnection *WsConnection) close() error {
	return wsConnection.conn.Close()
}

// MessageHandleFunc allows to add a handler for a given message type
func (wsConnection *WsConnection) MessageHandleFunc(messageType string, handleFunc func(WsMessage) (*WsMessage, error)) {
	wsConnection.messageHandlers[messageType] = handleFunc
}

// RequestDeserializer allows to register a deserializer function for the incoming request messages
func (wsConnection *WsConnection) RequestDeserializer(requestDeserializer func([]byte) (*WsMessageBody, error)) {
	wsConnection.requestDeserializer = requestDeserializer
}

// ResponseSerializer allows to register a serializer function for the outgoing response messages
func (wsConnection *WsConnection) ResponseSerializer(responseSerializer func(WsMessageBody) ([]byte, error)) {
	wsConnection.responseSerializer = responseSerializer
}

// ErrorHandler allows to register an error handler function to handle errors occuring during request processing
func (wsConnection *WsConnection) ErrorHandler(errorHandler func(error) WsMessageBody) {
	wsConnection.errorHandler = errorHandler
}

func (wsConnection *WsConnection) handleResponses() {
	for response := range wsConnection.responses {
		message, err := wsConnection.responseSerializer(response.Body)
		if err != nil {
			select {
			case <-wsConnection.errors:
				log.Printf("No longer sending error [%s] as connection was closed in the meantime.\n", err)
				return
			default:
				wsConnection.errors <- wsErrorMessage{Type: response.Type, Error: err}
			}
		} else {
			wsConnection.conn.WriteMessage(response.Type, message)
		}
	}
}

func (wsConnection *WsConnection) handleErrors() {
	for error := range wsConnection.errors {
		message, err := wsConnection.responseSerializer(wsConnection.getErrorHandler()(error.Error))
		if err != nil {
			log.Printf("Unexpected error serializing error message [%v][%s]\n", error, err)
			message = []byte("Unexpected error")
		}
		wsConnection.conn.WriteMessage(error.Type, message)
	}
}

func (wsConnection *WsConnection) handleRequests() {
	for {
		messageType, message, err := wsConnection.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			wsConnection.closeChannels()
			break
		} else {
			log.Printf("message: %s", message)
			body, err := wsConnection.requestDeserializer(message)
			if err != nil {
				wsConnection.errors <- wsErrorMessage{Type: messageType, Error: err}
			} else {
				wsConnection.requests <- WsMessage{Type: messageType, Body: *body}
			}
		}
	}
}

func (wsConnection *WsConnection) processRequests() {
	for request := range wsConnection.requests {
		log.Printf("recv: %s", request.Body)
		go func(request WsMessage) {
			if !wsConnection.hasMessageHandler(request.Body.Type) {
				select {
				case <-wsConnection.errors:
					log.Println("No longer sending error for missing message handler because connection was closed in the meantime.")
				default:
					wsConnection.errors <- wsErrorMessage{Type: request.Type, Error: fmt.Errorf("No handler for request type %s", request.Body.Type)}
				}
				return
			}

			message, err := wsConnection.messageHandlers[request.Body.Type](request)
			select {
			case <-wsConnection.responses: // if responses is closed then errors is also closed, so not checking that
				log.Printf("No longer sending response/error [%v / %s] because connection was closed in the meantime.\n", message, err)
				return
			default:
				if err != nil {
					wsConnection.errors <- wsErrorMessage{Type: request.Type, Error: err}
				} else {
					wsConnection.responses <- *message
				}
			}
		}(request)
	}
}

func (wsConnection *WsConnection) hasMessageHandler(messageType string) bool {
	_, messageHandlerFound := wsConnection.messageHandlers[messageType]
	return messageHandlerFound
}

func (wsConnection *WsConnection) getErrorHandler() func(error) WsMessageBody {
	if wsConnection.errorHandler != nil {
		return wsConnection.errorHandler
	}
	return defaultErrorHandler
}

func defaultErrorHandler(err error) WsMessageBody {
	return WsMessageBody{Type: "error", Content: err.Error()}
}

func (wsConnection *WsConnection) closeChannels() {
	close(wsConnection.requests)
	close(wsConnection.responses)
	close(wsConnection.errors)
}
