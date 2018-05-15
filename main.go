package main

import (
	"fmt"
	"log"
	"net/http"

	r "gopkg.in/gorethink/gorethink.v4"
)

func main() {
	fmt.Println("[Rethink] Connecting...")
	session, err := r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "slacker",
	})
	if err != nil {
		log.Panic(err.Error())
	}

	fmt.Println("[Router] Creating router...")
	router := NewRouter(session)

	fmt.Println("[Handlers] Registering handlers...")

	// Rooms
	router.Handle("room add", addRoom)
	router.Handle("room subscribe", subscribeRoom)
	router.Handle("room unsubscribe", unsubscribeRoom)

	// Users
	// router.Handle("user add", addUser)
	router.Handle("user edit", editUser)
	router.Handle("user subscribe", subscribeUser)
	router.Handle("user unsubscribe", unsubscribeUser)

	//Messages
	router.Handle("message add", addMessage)
	router.Handle("message subscribe", subscribeMessage)
	router.Handle("message unsubscribe", unsubscribeMessage)

	http.Handle("/", router)
	fmt.Println("[WWW] Listening on port 4000...")
	http.ListenAndServe(":4000", nil)
}
