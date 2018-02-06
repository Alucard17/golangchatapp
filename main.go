package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Message struct {
	From         string `json="from"`
	To           string `json="to"`
	Json_message string `json="json_message"`
}

var clientlist = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	var username = r.URL.Query().Get("username")
	clientlist[username] = conn
	for {
		message := Message{From: username}
		if err := conn.ReadJSON(&message); err != nil {
			conn.Close()
			delete(clientlist, username)
			panic(err)
		}

		fmt.Println(message)
		broadcast <- message
	}

}

func usershandler(w http.ResponseWriter, r *http.Request) {
	var userlist []string
	for k, _ := range clientlist {
		userlist = append(userlist, k)
	}
	data, err := json.Marshal(userlist)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func messageworker() {
	for {
		msg := <-broadcast
		client1 := clientlist[msg.From]
		err := client1.WriteJSON(msg)
		if err != nil {
			log.Printf("error: %v", err)
			client1.Close()
			delete(clientlist, msg.From)
		}

		client2 := clientlist[msg.To]
		err = client2.WriteJSON(msg)
		if err != nil {
			log.Printf("error: %v", err)
			client2.Close()
			delete(clientlist, msg.To)
		}

	}
}

func main() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handler)
	http.HandleFunc("/getUserList", usershandler)
	go messageworker()
	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
