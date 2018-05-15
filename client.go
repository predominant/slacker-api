package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	r "gopkg.in/gorethink/gorethink.v4"
)

type FindHandler func(string) (Handler, bool)

type Client struct {
	send         chan Message
	socket       *websocket.Conn
	findHandler  FindHandler
	session      *r.Session
	stopChannels map[int]chan bool
	id           string
	username     string
}

func (client *Client) NewStopChannel(key int) chan bool {
	client.StopForKey(key)
	stop := make(chan bool)
	client.stopChannels[key] = stop
	return stop
}

func (client *Client) StopForKey(key int) {
	if ch, found := client.stopChannels[key]; found {
		ch <- true
		delete(client.stopChannels, key)
	}
}

func (client *Client) Read() {
	var message Message
	for {
		if err := client.socket.ReadJSON(&message); err != nil {
			break
		}
		if handler, found := client.findHandler(message.Name); found {
			handler(client, message.Data)
		}
	}
	client.socket.Close()
}

func (client *Client) Write() {
	for msg := range client.send {
		if err := client.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	client.socket.Close()
}

func (client *Client) Close() {
	fmt.Print("Closing client connection: ")
	fmt.Println(client.id)
	for _, ch := range client.stopChannels {
		ch <- true
	}
	close(client.send)
	r.Table("user").
		Get(client.id).
		Delete().
		Exec(client.session)
}

func NewClient(socket *websocket.Conn, findHandler FindHandler, session *r.Session) *Client {
	var user User
	user.Name = "anonymous"
	result, err := r.Table("user").
		Insert(user).
		RunWrite(session)
	if err != nil {
		log.Println(err.Error())
	}
	var id string
	if len(result.GeneratedKeys) > 0 {
		id = result.GeneratedKeys[0]
	}
	fmt.Print("New client connection: ")
	fmt.Print(id)
	return &Client{
		send:         make(chan Message),
		socket:       socket,
		findHandler:  findHandler,
		session:      session,
		stopChannels: make(map[int]chan bool),
		id:           id,
		username:     user.Name,
	}
}
