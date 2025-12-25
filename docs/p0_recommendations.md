# P0 Recommendations: Sprint 4 Optimization & Refactoring

> **Context:** Post-Alpha 0.5. The system is functional but tightly coupled. To scale and ensure stability (fix deadlocks, improve physics), we must refactor before adding more features.

## 🚨 P0: Critical Architecture Decoupling (High Priority)

**Current State:**
The `Room` struct in `network` package handles both:
1.  WebSocket Lifecycle (Client Register/Unregister/Broadcast).
2.  Game Simulation Loop (Ticker, Physics Update, Interaction).

**Problem:**
-   **Lock Contention:** `Room` holds locks for network ops, blocking the game loop.
-   **Deadlocks:** Interaction logic calling `RemovePlayer` inside `UpdateTick` caused deadlocks.
-   **Testing:** Cannot test game logic without mocking a network connection.

**Recommendation:**
Split into two distinct components communicating via Channels:
1.  **`GameLoop` (in `logic` package):**
    -   Owns `GameState`.
    -   Runs the Physics Ticker.
    -   Consumes `InputEvents` from a channel.
    -   Produces `StateSnapshots` to a channel.
    -   *Pure logic, no networking.*
2.  **`Room` (in `network` package):**
    -   Owns `Clients` map.
    -   Owns `GameLoop` instance.
    -   Forwards Client messages -> `GameLoop.InputChan`.
    -   Consumes `GameLoop.SnapshotChan` -> Broadcasts to Clients.

## 🚨 P0: Physics Engine Upgrade

**Current State:**
-   `isWalkableWithRadius` checks 4 points around the player against the grid map.
-   Result: Players get "stuck" on corners or have jagged movement.

**Recommendation:**
-   Implement **AABB (Axis-Aligned Bounding Box)** vs **Tile AABB** collision.
-   Or **Circle vs AABB** collision (since players are visually circular).
-   **Slide Vector:** When colliding, project velocity onto the wall plane to allow "sliding" instead of stopping dead.

## 🚨 P1: Protocol Optimization (Protobuf)

**Current State:**
-   JSON serialization (`encoding/json`) is CPU intensive and verbose.
-   Bandwidth usage is high for 20Hz updates.

**Recommendation:**
-   Define `.proto` schema for `StateSnapshot` and `InputEvent`.
-   Generate Go code.
-   Replace `SendJSON` with binary Protobuf payloads.

---

**Execution Plan:**
1.  **Refactor `GameLoop`** (Decoupling) - *Immediate Action*.
2.  **Upgrade Physics** - *Follow-up*.
3.  **Protobuf** - *Final Step*.