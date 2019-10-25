package websocket

import (
	"encoding/json"
)

type jsonWsMessageBody struct {
	Type    string      `json:"type"`
	Content interface{} `json:"body"`
}

// JSONRequestDeserializer can be used as a default request deserializer for incoming json messages
func JSONRequestDeserializer(message []byte) (*WsMessageBody, error) {
	var jsonWsMessageBody jsonWsMessageBody
	err := json.Unmarshal(message, &jsonWsMessageBody)
	if err != nil {
		return nil, err
	}
	return &WsMessageBody{Type: jsonWsMessageBody.Type, Content: jsonWsMessageBody.Content}, nil
}

// JSONResponseSerializer can be used as a default response serializer for outgoing json messages
func JSONResponseSerializer(messageBody WsMessageBody) ([]byte, error) {
	res, err := json.Marshal(messageBody)
	if err != nil {
		return nil, err
	}
	return res, nil
}
