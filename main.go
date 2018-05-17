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

	fmt.Println(
		"[Rethink] Connecting...",
		fmt.Sprintf("%s:%d", config.Database.Host, config.Database.Port),
		"/",
		config.Database.Name)
	session, err := r.Connect(r.ConnectOpts{
		Address:  fmt.Sprintf("%s:%d", config.Database.Host, config.Database.Port),
		Database: config.Database.Name,
	})
	if err != nil {
		log.Panic(err.Error())
	}

	// Create the database(if not existing already)
	err = r.DBList().
		Contains(config.Database.Name).
		Do(func(dbExists r.Term) r.Term {
			return r.Branch(
				dbExists,
				map[string]uint{"dbs_created": 0},
				r.DBCreate(config.Database.Name))
		}).
		Exec(session)
	if err != nil {
		log.Panic(err.Error())
	}

	// Create tables
	tables := []string{"user", "room", "message"}
	for _, tableName := range tables {
		err = r.TableList().
			Contains(tableName).
			Do(func(tableExists r.Term) r.Term {
				return r.Branch(
					tableExists,
					map[string]uint{"tables_created": 0},
					r.TableCreate(tableName))
			}).
			Exec(session)
		if err != nil {
			log.Panic(err.Error())
		}
	}

	// Create Default Room
	err = r.Table("room").
		Filter(map[string]bool{"default": true}).
		Count().
		Do(func(numTables r.Term) r.Term {
			return r.Branch(
				r.Ge(numTables, 1),
				map[string]uint{"something": 0},
				r.Table("room").Insert(map[string]interface{}{
					"name":    "general",
					"default": true,
				}))
		}).
		Exec(session)
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
