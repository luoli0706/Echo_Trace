package logic

import (
	"log"
	"time"
)

type InputType int

const (
	InputMove InputType = iota
	InputUseItem
	InputInteract
	InputPickup
	InputDrop
	InputSell
	InputBuy
	InputTactic
	InputLogin
	InputDevSkip
)

type PlayerInput struct {
	SessionID string
	Type      InputType
	// Payload fields (can be generic or specific)
	Dir        Vector2
	LookDir    Vector2
	HasLookDir bool
	SlotIndex  int
	ItemID     string
	Tactic     string
	Name       string
}

type GameLoop struct {
	GameState    *GameState
	InputChan    chan PlayerInput
	SnapshotChan chan map[string]interface{} // map[sessionID]snapshot
	StopChan     chan bool
}

func NewGameLoop(cfg *GameConfig) *GameLoop {
	return &GameLoop{
		GameState:    NewGameState(cfg),
		InputChan:    make(chan PlayerInput, 100),
		SnapshotChan: make(chan map[string]interface{}), // Unbuffered? Or 1?
		StopChan:     make(chan bool),
	}
}

func (gl *GameLoop) Run() {
	ticker := time.NewTicker(time.Duration(gl.GameState.Config.Server.TickRateMs) * time.Millisecond)
	defer ticker.Stop()

	log.Println("GameLoop Started.")

	for {
		select {
		case input := <-gl.InputChan:
			gl.handleInput(input)

		case <-ticker.C:
			// 1. Physics & Logic Update
			dt := float64(gl.GameState.Config.Server.TickRateMs) / 1000.0
			gl.GameState.UpdateTick(dt)

			// 2. Generate Snapshots
			// We need to generate snapshots for ALL connected players (alive or dead or extracted)
			// But GameState doesn't know who is "connected" if they are removed?
			// Actually GameState.Players contains all *active* players.
			// What about spectating players?
			// In our previous fix, Extracted players remain in GameState.Players.
			// So iterating gs.Players is enough.

			snapshots := make(map[string]interface{})

			// We need a read lock here? UpdateTick releases lock when done.
			// GetSnapshot grabs RLock.

			// Optimization: Compute shared data once?
			// For now, iterate keys.
			// We need to know SessionIDs to generate for.
			// GameState.Players has them.

			gl.GameState.Mutex.RLock()
			ids := make([]string, 0, len(gl.GameState.Players))
			for id := range gl.GameState.Players {
				ids = append(ids, id)
			}
			gl.GameState.Mutex.RUnlock()

			for _, id := range ids {
				snap := gl.GameState.GetSnapshot(id)
				if snap != nil {
					snapshots[id] = snap
				}
			}

			// Send to Network Layer
			// Non-blocking send to avoid stalling loop if network is slow?
			// Or blocking to ensure sync?
			select {
			case gl.SnapshotChan <- snapshots:
			default:
				// Skip frame if network is busy
			}

		case <-gl.StopChan:
			log.Println("GameLoop Stopped.")
			return
		}
	}
}

func (gl *GameLoop) handleInput(input PlayerInput) {
	// These methods already acquire Lock inside GameState
	// If we move to pure Actor model, GameState methods shouldn't lock, GameLoop should own the lock.
	// But for now, let's keep locks in GameState to minimize refactor risk,
	// just serialize calls here.

	gs := gl.GameState
	sid := input.SessionID

	switch input.Type {
	case InputMove:
		gs.HandleInput(sid, input.Dir, input.LookDir, input.HasLookDir)
	case InputUseItem:
		gs.HandleUseItem(sid, input.SlotIndex)
	case InputInteract:
		gs.HandleInteract(sid)
	case InputPickup:
		gs.HandlePickup(sid)
	case InputDrop:
		gs.HandleDropItem(sid, input.SlotIndex)
	case InputSell:
		gs.HandleSellItem(sid, input.SlotIndex)
	case InputBuy:
		gs.HandleBuyItem(sid, input.ItemID)
	case InputTactic:
		gs.HandleChooseTactic(sid, input.Tactic)
	case InputLogin:
		gs.SetPlayerName(sid, input.Name)
	case InputDevSkip:
		gs.HandleDevSkipPhase()
	}
}
