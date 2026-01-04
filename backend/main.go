package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"echo_trace_server/logic"
	"echo_trace_server/network"
	"echo_trace_server/storage"
)

var Config logic.GameConfig

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// 0. Init Persistence
	storage.InitDB("./game.db")

	// 0.1 Init Network Manager
	network.InitManager()

	// 1. Load Config (Default)
	// In dev we run from ./backend (configs live in ../config),
	// but in Docker we run from /app (configs live in ./config).
	wd, _ := os.Getwd()
	candidates := []string{}
	if wd != "" {
		candidates = append(candidates, wd)
		candidates = append(candidates, filepath.Dir(wd))
	}
	if len(candidates) == 0 {
		candidates = append(candidates, ".")
	}

	rootDir := ""
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "config")); err == nil {
			rootDir = abs
			break
		}
		if _, err := os.Stat(filepath.Join(abs, "game_config.json")); err == nil {
			rootDir = abs
			break
		}
	}
	if rootDir == "" {
		rootDir, _ = filepath.Abs(".")
	}

	cfg, err := logic.LoadDefaultGameConfig(rootDir)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	Config = cfg

	// Enforce safety bounds on loaded defaults.
	logic.ClampGameConfig(&Config)

	// Make loaded defaults available to room creation.
	network.SetDefaultConfig(&Config)

	// 3. Router Setup
	mux := http.NewServeMux()

	// WebSocket Endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.ServeWs(w, r)
	})

	// Health Check Endpoint (For future load balancers/k8s)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 4. Start Server
	addr := ":" + *port
	log.Printf("Echo Trace Server listening on %s", addr)

	// Use the mux
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
