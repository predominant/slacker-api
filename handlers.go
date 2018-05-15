package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	r "gopkg.in/gorethink/gorethink.v4"
)

const (
	RoomStop = iota
	UserStop
	MessageStop
)

type Message struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type ChannelMessage struct {
	Id        string    `json:"id" gorethink:"id,omitempty"`
	RoomId    string    `json:"roomId" gorethink:"roomId"`
	Body      string    `json:"body" gorethink:"body"`
	Author    string    `json:"author" gorethink:"author"`
	CreatedAt time.Time `json:"createdAt" gorethink:"createdAt"`
}

type User struct {
	ID   string `json:"id" gorethink:"id,omitempty"`
	Name string `json:"name" gorethink:"name"`
}

type Room struct {
	ID   string `json:"id" gorethink:"id,omitempty"`
	Name string `json:"name" gorethink:"name"`
}

func addRoom(client *Client, data interface{}) {
	var room Room
	err := mapstructure.Decode(data, &room)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	go func() {
		err = r.Table("room").
			Insert(room).
			Exec(client.session)
		if err != nil {
			client.send <- Message{"error", err.Error()}
			return
		}
	}()
}

func subscribeRoom(client *Client, data interface{}) {
	stop := client.NewStopChannel(RoomStop)
	result := make(chan r.ChangeResponse)

	cursor, err := r.Table("room").
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Run(client.session)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	go func() {
		var change r.ChangeResponse
		for cursor.Next(&change) {
			result <- change
		}
	}()

	go func() {
		for {
			select {
			case <-stop:
				cursor.Close()
				return
			case change := <-result:
				if change.NewValue != nil && change.OldValue == nil {
					// Insert (new room)
					client.send <- Message{"room add", change.NewValue}
					fmt.Println("send room add msg")
				}
			}
		}
	}()
}

func unsubscribeRoom(client *Client, data interface{}) {
	client.StopForKey(RoomStop)
}

func editUser(client *Client, data interface{}) {
	var user User
	err := mapstructure.Decode(data, &user)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}
	client.username = user.Name

	fmt.Printf("Handler: editUser (username = %s)\n", client.username)

	go func() {
		_, err = r.Table("user").
			Get(client.id).
			Update(user).
			RunWrite(client.session)
		if err != nil {
			client.send <- Message{"error", err.Error()}
		}
	}()
}

func subscribeUser(client *Client, data interface{}) {
	stop := client.NewStopChannel(UserStop)
	result := make(chan r.ChangeResponse)

	cursor, err := r.Table("user").
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Run(client.session)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	go func() {
		var change r.ChangeResponse
		for cursor.Next(&change) {
			result <- change
		}
	}()

	go func() {
		for {
			select {
			case <-stop:
				cursor.Close()
				return
			case change := <-result:
				if change.NewValue != nil && change.OldValue == nil {
					// Insert (new user)
					client.send <- Message{"user add", change.NewValue}
					fmt.Println("send user add msg")
				}
				if change.NewValue != nil && change.OldValue != nil {
					// Update (changed user)
					client.send <- Message{"user edit", change.NewValue}
					fmt.Println("send user edit msg")
				}
				if change.NewValue == nil && change.OldValue != nil {
					// Delete (removed user)
					client.send <- Message{"user remove", change.OldValue}
					fmt.Println("send user remove msg")
				}
			}
		}
	}()
}

func unsubscribeUser(client *Client, data interface{}) {

}

func addMessage(client *Client, data interface{}) {
	fmt.Println("Handler: addMessage")
	var channelMessage ChannelMessage

	err := mapstructure.Decode(data, &channelMessage)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	go func() {
		channelMessage.Body = strings.TrimSpace(channelMessage.Body)
		if len(channelMessage.Body) == 0 {
			fmt.Println("Discarding empty message")
			return
		}
		channelMessage.CreatedAt = time.Now()
		channelMessage.Author = client.username
		fmt.Printf("%#v\n", channelMessage)
		err = r.Table("message").
			Insert(channelMessage).
			Exec(client.session)
		if err != nil {
			client.send <- Message{"error", err.Error()}
			return
		}
	}()
}

func subscribeMessage(client *Client, data interface{}) {
	// Should pass in a room Id
	eventData := data.(map[string]interface{})
	val, ok := eventData["roomId"]
	if !ok {
		return
	}
	roomId, ok := val.(string)
	if !ok {
		return
	}

	stop := client.NewStopChannel(MessageStop)
	result := make(chan r.ChangeResponse)

	cursor, err := r.Table("message").
		OrderBy(r.OrderByOpts{Index: r.Desc("createdAt")}).
		Filter(r.Row.Field("roomId").Eq(roomId)).
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Run(client.session)
	if err != nil {
		client.send <- Message{"error", err.Error()}
		return
	}

	go func() {
		var change r.ChangeResponse
		for cursor.Next(&change) {
			result <- change
		}
	}()

	go func() {
		for {
			select {
			case <-stop:
				cursor.Close()
				return
			case change := <-result:
				if change.NewValue != nil && change.OldValue == nil {
					var channelMessage ChannelMessage
					mapstructure.Decode(data, &channelMessage)

					// Insert (new message)
					// fmt.Printf("%#v\n", channelMessage)
					client.send <- Message{"message add", change.NewValue}
					fmt.Println("send message add msg")
				}
			}
		}
	}()
}

func unsubscribeMessage(client *Client, data interface{}) {
	client.StopForKey(MessageStop)
}
