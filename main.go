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
var broadcast = make(chan *Broadcast)
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

// Broadcast keeps track of which client the message is from
type Broadcast struct {
	Client *websocket.Conn
	Msg    *Message
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

	go handleMessages(conn)
}

func handleMessages(client *websocket.Conn) {
	for {
		if _, ok := clients[client]; !ok {
			// Stop reading if client is closed
			break
		}
		// receive message
		msg := &Message{}
		err := client.ReadJSON(msg)
		if err != nil {
			client.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "INVALID FORMAT"),
				time.Now().Add(1000*time.Millisecond))
			client.Close()
			break
		}

		// Add the msg to the broadcast channel
		broadcast <- &Broadcast{Client: client, Msg: msg}
	}
}

func replyMessages() {
	for {
		// Grab the next message from the broadcast channel
		bc := <-broadcast
		msg := bc.Msg

		if _, ok := clients[bc.Client]; !ok {
			// Stop writing if client is closed
			break
		}

		fmt.Println(msg.Action)
		if msg.Action == "HEARTBEAT" {
			msg.Action = "HEARTBEAT_ACK"
			// Don't want every client to receive HEARTBEAT_ACK, only the one that sent it
			err := bc.Client.WriteJSON(msg)
			if err != nil {
				bc.Client.Close()
				log.Fatal(err)
			}
		} else if msg.Action == "MESSAGE_CREATE" {
			if len(msg.Data.Content) > 2000 {
				bc.Client.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "MESSAGE TOO LONG"),
					time.Now().Add(1000*time.Millisecond))
				bc.Client.Close()
				break
			}
			for client := range clients {
				// Write to the websocket
				err := client.WriteJSON(msg)
				if err != nil {
					client.Close()
					log.Fatal(err)
				}
			}
		} else {
			bc.Client.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, "INVALID ACTION"),
				time.Now().Add(1000*time.Millisecond))
			bc.Client.Close()
			break
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
	go replyMessages()

	fmt.Println("http server started on :8000")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
