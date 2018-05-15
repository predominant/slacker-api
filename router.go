package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	r "gopkg.in/gorethink/gorethink.v4"
)

type Handler func(*Client, interface{})

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Router struct {
	rules   map[string]Handler
	session *r.Session
}

func NewRouter(session *r.Session) *Router {
	fmt.Println("[Router] Creating new router...")
	return &Router{
		rules:   make(map[string]Handler),
		session: session,
	}
}

func (r *Router) Handle(msgName string, handler Handler) {
	fmt.Printf("[Router] Handler registered: %s\n", msgName)
	r.rules[msgName] = handler
}

func (r *Router) FindHandler(msgName string) (Handler, bool) {
	handler, found := r.rules[msgName]
	return handler, found
}

func (e *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[WWW] Serving HTTP")
	fmt.Println("[WWW] Upgrading...")
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	fmt.Println("[WWW] Creating client...")
	client := NewClient(socket, e.FindHandler, e.session)
	defer client.Close()
	go client.Write()
	client.Read()
}
