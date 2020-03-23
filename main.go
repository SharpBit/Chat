package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan *Message)
var upgrader = websocket.Upgrader{}

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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	clients[conn] = true

	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	conn.SetCloseHandler(func(code int, text string) error {
		delete(clients, conn)
		return nil
	})

	for {
		// receive message
		msg := &Message{}
		err = conn.ReadJSON(msg)
		if err != nil {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "INVALID FORMAT"),
				time.Now().Add(1000*time.Millisecond))
			conn.Close()
			break
		}

		// Add the msg to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			fmt.Println(msg.Action)
			if msg.Action == "HEARTBEAT" {
				msg.Action = "HEARTBEAT_ACK"
			} else if msg.Action == "MESSAGE_CREATE" {
				if len(msg.Data.Content) > 2000 {
					client.WriteControl(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "MESSAGE TOO LONG"),
						time.Now().Add(1000*time.Millisecond))
					client.Close()
					break
				}
			} else {
				client.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, "INVALID ACTION"),
					time.Now().Add(1000*time.Millisecond))
				client.Close()
				break
			}

			// Write to the websocket
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				log.Fatal(err)
			}
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Handle static dirs
	static := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", static))
	scripts := http.FileServer(http.Dir("./scripts"))
	http.Handle("/scripts/", http.StripPrefix("/scripts/", scripts))

	// Handle Views
	http.HandleFunc("/", index)
	http.HandleFunc("/ws", ws)
	go handleMessages()

	fmt.Println("http server started on :8000")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
