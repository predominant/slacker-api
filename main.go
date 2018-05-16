package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/BurntSushi/toml"
	r "gopkg.in/gorethink/gorethink.v4"
)

// var Config = struct {
// 	Port     uint `default:"4000"`
// 	Database struct {
// 		Host string `default:"localhost"`
// 		Port uint   `default:"28015"`
// 		Name string `default:"slacker"`
// 	}
// }{}

func main() {
	configFilePtr := flag.String("config", "config.toml", "Configuration file")
	flag.Parse()
	configFile := *configFilePtr

	type DatabaseConfig struct {
		Host string
		Port uint
		Name string
	}
	type Config struct {
		Port     uint
		Database DatabaseConfig
	}

	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Panic(err.Error())
	}

	config := Config{}
	toml.Unmarshal(configData, &config)

	fmt.Println("[Rethink] Connecting...")
	session, err := r.Connect(r.ConnectOpts{
		Address:  fmt.Sprintf("%s:%d", config.Database.Host, config.Database.Port),
		Database: config.Database.Name,
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
	fmt.Println(fmt.Sprintf("[WWW] Listening on port %d...", config.Port))
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
