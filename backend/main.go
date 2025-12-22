package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"echo_trace_server/logic"
	"echo_trace_server/network"
)

var Config logic.GameConfig

func main() {
	// Load Config
	absPath, _ := filepath.Abs("../game_config.json")
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if err := json.Unmarshal(data, &Config); err != nil {
		log.Fatalf("Parse config error: %v", err)
	}

	// Init Room
	globalRoom := network.NewRoom("alpha_1", &Config)
	go globalRoom.Run()

	// HTTP Handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.ServeWs(globalRoom, w, r)
	})

	addr := ":8080"
	log.Printf("Echo Trace Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
