package main

import (
	"os"
	"log"
	"fmt"
	"time"
	"sync"
	"net/http"
	"encoding/json"
	"restreamer"
)
//	"encoding/hex"

type Configuration struct {
	Listen string
	Timeout int
	Buffer int
	Streams []struct {
		Serve string
		Remote string
	}
}

func main() {
	var configname string
	if len(os.Args) > 1 {
		configname = os.Args[1]
	} else {
		configname = "server.json"
	}
	
	configfile, err := os.Open(configname)
	if err != nil {
		log.Fatal("Can't read configuration from server.json\n", err)
	}
	decoder := json.NewDecoder(configfile)
	config := Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Error parsing configuration\n", err)
	}
	configfile.Close()

	log.Printf("Listen = %s\n", config.Listen)
	log.Printf("Timeout = %d\n", config.Timeout)

	i := 0
	mux := http.NewServeMux()
	for _, streamdef := range config.Streams {
		log.Printf("Testing stream %s\n", streamdef.Remote)
		client := restreamer.NewHttpClient(m.Remote)
		err := client.Connect()
		if err == nil {
			mux.Handle(streamdef.Serve, client)
			log.Printf("Handled connection %d", i)
			i++
		} else {
			log.Print(err)
		}
	}
	
	if i == 0 {
		log.Fatal(HttpClientError { "No streams available" })
	} else {
		log.Print("Starting server")
		server := &http.Server {
			Addr: config.Listen,
			Handler: mux,
			ReadTimeout: time.Duration(config.Timeout) * time.Second,
			WriteTimeout: time.Duration(config.Timeout) * time.Second,
			//MaxHeaderBytes: 1 << 20,
		}
		log.Fatal(server.ListenAndServe())
	}
}
