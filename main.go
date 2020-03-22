package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var sockets = make(map[string]*websocket.Conn)

// Message received by or sent to server
type Message struct {
	Action string      `json:"action"`
	Data   MessageData `json:"data"`
}

// MessageData is the data for the message
type MessageData struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

func index(w http.ResponseWriter, r *http.Request) {
	bytes, err := ioutil.ReadFile("views/index.html")
	if err != nil {
		fmt.Print(err)
	}

	fmt.Fprint(w, string(bytes))
}

func ws(w http.ResponseWriter, r *http.Request) {
	u := websocket.Upgrader{}
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	conn.SetReadLimit(2000)

	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	conn.SetCloseHandler(func(code int, text string) error {
		return conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "SOCKET TIMEOUT"),
			time.Now().Add(1000*time.Millisecond))
	})

	// receive message
	msg := &Message{}
	err = conn.ReadJSON(msg)
	if err != nil {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "INVALID FORMAT"),
			time.Now().Add(1000*time.Millisecond))
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println(msg.Action)
	if msg.Action == "HEARTBEAT" {
		msg.Action = "HEARTBEAT_ACK"
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Fatal(err)
		}
	} else if msg.Action == "MESSAGE_CREATE" {
		if len(msg.Data.Content) > 2000 {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "MESSAGE TOO LONG"),
				time.Now().Add(1000*time.Millisecond))
		}

		err = conn.WriteJSON(msg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, "INVALID ACTION"),
			time.Now().Add(1000*time.Millisecond))
	}
}

func main() {
	// Handle static dirs
	static := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", static))
	scripts := http.FileServer(http.Dir("./scripts"))
	http.Handle("/scripts/", http.StripPrefix("/scripts/", scripts))

	// Handle Views
	http.HandleFunc("/", index)
	http.HandleFunc("/ws", ws)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
