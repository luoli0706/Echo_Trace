import threading
import websocket
import json
import time


class NetworkClient:
    def __init__(self, url, recv_queue, session_id=None, player_name=None):
        self.url = url
        self.recv_queue = recv_queue
        self.ws = None
        self.running = True
        self.thread = threading.Thread(target=self._run)
        self.thread.daemon = True
        self.connected = False

        self._lock = threading.Lock()
        self.session_id = session_id or ""
        self.player_name = player_name or ""
        self.auto_join_room_id = ""

    def start(self):
        self.thread.start()

    def _run(self):
        while self.running:
            try:
                print(f"Connecting to {self.url}...")
                self.ws = websocket.WebSocketApp(
                    self.url,
                    on_open=self._on_open,
                    on_message=self._on_message,
                    on_error=self._on_error,
                    on_close=self._on_close
                )
                self.ws.run_forever()
                time.sleep(3) 
            except Exception as e:
                print(f"Network error: {e}")
                time.sleep(3)

    def _on_open(self, ws):
        print("Connected to Server")
        self.connected = True

        # If we were previously in a room, auto re-join on reconnect.
        with self._lock:
            room_id = self.auto_join_room_id
            sid = self.session_id
            nm = self.player_name

        if room_id:
            payload = {"room_id": room_id}
            if sid:
                payload["session_id"] = sid
            if nm:
                payload["name"] = nm
            self.send({"type": 1011, "payload": payload})

    def _on_message(self, ws, message):
        try:
            data = json.loads(message)
            self.recv_queue.put(data)
        except Exception as e:
            print(f"JSON Parse Error: {e}")

    def _on_error(self, ws, error):
        print(f"WS Error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        print("Disconnected")
        self.connected = False

    def send(self, data):
        if self.ws and self.connected:
            try:
                self.ws.send(json.dumps(data))
            except Exception as e:
                print(f"Send Error: {e}")

    def set_identity(self, session_id=None, player_name=None):
        with self._lock:
            if session_id is not None:
                self.session_id = str(session_id)
            if player_name is not None:
                self.player_name = str(player_name)

    def set_auto_join(self, room_id):
        with self._lock:
            self.auto_join_room_id = str(room_id or "")

    def clear_auto_join(self):
        with self._lock:
            self.auto_join_room_id = ""
