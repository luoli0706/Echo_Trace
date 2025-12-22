# Echo Trace (DarkForest-Go) Alpha

> **Latest Update:** v0.4 - Phase Logic, Motors, and Enhanced UI.

Echo Trace is a high-performance backend game demo featuring **Maze Scavenging + AOI Fog of War + Extraction Mechanics**.
Built with **Golang** (Server) and **Python/Pygame** (Client).

## 📂 Directory Structure
```
Echo_Trace/
├── backend/            # Golang Server
│   ├── logic/          # Core Logic (Physics, Maze, AOI, Items)
│   ├── network/        # WebSocket & Room Management
│   └── main.go         # Entry Point
├── frontend/           # Python Client
│   ├── client/         # Client Modules (Net, Render, State)
│   └── main.py         # Entry Point
├── game_config.json    # Shared Parameters
├── protocol.json       # Network Protocol Schema
└── README.md           # Documentation
```

## 🚀 Quick Start

### 1. Start Server
Requires Go 1.18+.
```bash
cd backend
go mod tidy
go run main.go
```
*Listens on :8080 by default.*

### 2. Start Client
Requires Python 3.10+.
```bash
cd frontend
pip install pygame-ce websocket-client
python main.py
```
*Open multiple terminals to simulate multiple players.*

## 🎮 Gameplay Guide

### Controls
*   **WASD:** Move Character (🏃)
*   **E:** Pick up Item (📦)
*   **F:** Interact / Fix Motor (⚡) (Hold to fix)
*   **Space:** Melee Attack / Use Weapon
*   **1-6:** Use Inventory Item
*   **Mouse Click:** UI Interaction (Settings ⚙️)

### Phases
1.  **SEARCH (0-120s):** Scavenge for items in the dark.
2.  **CONFLICT:** Motors (⚡) appear. Fix 5 motors or kill rivals.
    *   *New:* Motors pulse every 15s to reveal location.
3.  **ESCAPE:** The Exit (🚪) opens. Reach it to win.

### Features
*   **AOI Fog of War:** You only see what's physically visible to you.
*   **Physics:** Smooth wall-sliding collision detection (Radius: 0.25).
*   **Items:** Offense (Red), Survival (Green), Recon (Blue).
*   **UI:** Real-time HP bars, Phase Timer, System Clock, and Item Encyclopedia.

## 🛠 Tech Stack
*   **Server:** Go (Gorilla WebSocket), Mutex-protected GameState, Grid-based Map.
*   **Client:** Pygame CE, Interpolated Rendering, Cyberpunk UI style.
*   **Protocol:** JSON-over-WebSocket (Phase-driven state sync).