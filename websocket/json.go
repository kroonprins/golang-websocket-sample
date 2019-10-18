package websocket

import (
	"encoding/json"
	"log"
)

type JsonWsMessageBody struct {
	Type    string      `json:"type"`
	Content interface{} `json:"body"`
}

func JsonDeserializer(message []byte) WsMessageBody {
	var jsonWsMessageBody JsonWsMessageBody
	err := json.Unmarshal(message, &jsonWsMessageBody)
	if err != nil {
		log.Println(err)
	}
	return WsMessageBody{Type: jsonWsMessageBody.Type, Content: jsonWsMessageBody.Content}
}

func JsonSerializer(messageBody WsMessageBody) []byte {
	res, err := json.Marshal(messageBody)
	if err != nil {
		log.Println(err)
	}
	return res
}
