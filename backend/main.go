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
	// 1. Load Config
	absPath, _ := filepath.Abs("../game_config.json")
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if err := json.Unmarshal(data, &Config); err != nil {
		log.Fatalf("Parse config error: %v", err)
	}

	// 2. Init Room
	globalRoom := network.NewRoom("alpha_1", &Config)
	go globalRoom.Run()

	// 3. Router Setup
	mux := http.NewServeMux()
	
	// WebSocket Endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.ServeWs(globalRoom, w, r)
	})

	// Health Check Endpoint (For future load balancers/k8s)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 4. Start Server
	addr := ":8080"
	log.Printf("Echo Trace Server listening on %s", addr)
	
	// Use the mux
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}