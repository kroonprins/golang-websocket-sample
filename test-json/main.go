package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"
	"websocket/websocket"

	"github.com/mitchellh/mapstructure"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func echo(wsConnection *websocket.WsConnection) {
	wsConnection.RequestDeserializer(websocket.JsonDeserializer)
	wsConnection.ResponseSerializer(websocket.JsonSerializer)
	wsConnection.MessageHandleFunc("request1", messageHandler1)
	wsConnection.MessageHandleFunc("request2", messageHandler2)
	wsConnection.Listen()
}

func messageHandler1(request websocket.WsMessage) websocket.WsMessage {
	time.Sleep(2 * time.Second)
	requestMessage, err := decode(request.Body.Content)
	if err != nil {
		log.Println(err)
	}
	return websocket.WsMessage{Type: request.Type, Body: websocket.WsMessageBody{Type: "response1", Content: requestMessage.Message}}
}

type RequestMessage struct {
	Message string
}

func decode(message interface{}) (*RequestMessage, error) {
	var requestMessage RequestMessage
	err := mapstructure.Decode(message, &requestMessage)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &requestMessage, nil
}

func messageHandler2(request websocket.WsMessage) websocket.WsMessage {
	time.Sleep(7 * time.Second)
	return websocket.WsMessage{Type: request.Type, Body: websocket.WsMessageBody{Type: "response2", Content: request.Body.Content}}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", home)
	websocket.HandleFunc("/echo", echo, websocket.Upgrader{})
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send1").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input1.value);
        const message = {
            type: 'request1',
            body: {
                message: input1.value
            }
        }
        ws.send(JSON.stringify(message));
        return false;
	};
	document.getElementById("send2").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input2.value);
        const message = {
            type: 'request2',
            body: input2.value
        }
        ws.send(JSON.stringify(message));
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input1" type="text" value="Hello world 1">
<button id="send1">Send</button>
<p><input id="input2" type="text" value="Hello world 2">
<button id="send2">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
