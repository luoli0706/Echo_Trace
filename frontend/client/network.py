import threading
import websocket
import json
import time
import sys
import os

# 添加 proto 目录到路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
from proto import echo_trace_pb2 as pb


class NetworkClient:
    def __init__(self, url, recv_queue, session_id=None, player_name=None, connect_timeout_sec: float = 2.5):
        self.url = url
        self.recv_queue = recv_queue
        self.ws = None
        self.running = True
        self.thread = threading.Thread(target=self._run)
        self.thread.daemon = True
        self.connected = False
        self.last_error = ""

        try:
            self.connect_timeout_sec = float(connect_timeout_sec)
        except Exception:
            self.connect_timeout_sec = 2.5

        self._lock = threading.Lock()
        self.session_id = session_id or ""
        self.player_name = player_name or ""
        self.auto_join_room_id = ""

    def start(self):
        self.thread.start()

    def stop(self):
        self.running = False
        try:
            if self.ws is not None:
                self.ws.close()
        except Exception:
            pass

    def _run(self):
        while self.running:
            try:
                print(f"Connecting to {self.url}...")
                # Apply a default socket timeout so failed connects don't hang forever.
                try:
                    websocket.setdefaulttimeout(self.connect_timeout_sec)
                except Exception:
                    pass

                self.ws = websocket.WebSocketApp(
                    self.url,
                    on_open=self._on_open,
                    on_message=self._on_message,
                    on_error=self._on_error,
                    on_close=self._on_close
                )
                # Keep ping enabled so dead connections are detected.
                self.ws.run_forever(ping_interval=10, ping_timeout=5)
                # If the server closes immediately or connect fails, don't wait long to retry.
                time.sleep(1)
            except Exception as e:
                print(f"Network error: {e}")
                self.last_error = str(e)
                time.sleep(1)

    def _on_open(self, ws):
        print("Connected to Server")
        self.connected = True
        self.last_error = ""

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
            # 尝试解析为 JSON（文本消息）
            if isinstance(message, str):
                data = json.loads(message)
                self.recv_queue.put(data)
            # 二进制消息 -> Protobuf
            elif isinstance(message, bytes):
                envelope = pb.Envelope()
                envelope.ParseFromString(message)
                # 转换为字典格式供现有代码使用
                data = self._protobuf_to_dict(envelope)
                self.recv_queue.put(data)
        except Exception as e:
            print(f"Message Parse Error: {e}")

    def _on_error(self, ws, error):
        print(f"WS Error: {error}")
        self.last_error = str(error)

    def _on_close(self, ws, close_status_code, close_msg):
        print("Disconnected")
        self.connected = False
        if close_status_code or close_msg:
            self.last_error = f"closed ({close_status_code}): {close_msg}".strip()

    def send(self, data):
        """发送 JSON 消息（用于房间管理等）"""
        if self.ws and self.connected:
            try:
                self.ws.send(json.dumps(data))
            except Exception as e:
                print(f"Send Error: {e}")

    def send_binary(self, data):
        """发送二进制 Protobuf 消息（用于游戏数据）"""
        if self.ws and self.connected:
            try:
                self.ws.send(data, opcode=websocket.ABNF.OPCODE_BINARY)
            except Exception as e:
                print(f"Send Binary Error: {e}")

    def send_move(self, dir_x, dir_y, look_x, look_y):
        """发送移动指令（Protobuf）"""
        move_input = pb.C2S_MoveInput()
        move_input.dir.x = dir_x
        move_input.dir.y = dir_y
        if look_x is not None and look_y is not None:
            move_input.look_dir.x = look_x
            move_input.look_dir.y = look_y
        
        envelope = pb.Envelope()
        envelope.type = 2001  # MOVE_REQ
        envelope.payload = move_input.SerializeToString()
        self.send_binary(envelope.SerializeToString())

    def send_fire(self):
        """发送开火指令（Protobuf）"""
        envelope = pb.Envelope()
        envelope.type = 2010  # FIRE_REQ
        envelope.payload = b""  # 空载荷
        self.send_binary(envelope.SerializeToString())

    def _protobuf_to_dict(self, envelope):
        """将 Protobuf Envelope 转换为字典格式"""
        msg_type = envelope.type
        payload_bytes = envelope.payload
        
        # StateSnapshot (3001)
        if msg_type == 3001:
            snapshot = pb.S2C_StateSnapshot()
            snapshot.ParseFromString(payload_bytes)
            return {
                "type": msg_type,
                "payload": self._snapshot_to_dict(snapshot)
            }
        
        # 其他消息类型暂时保留原始处理
        return {"type": msg_type, "payload": {}}

    def _snapshot_to_dict(self, snapshot):
        """将 StateSnapshot 转换为字典"""
        return {
            "phase": snapshot.phase,
            "time_left": snapshot.time_left,
            "total_kills": snapshot.total_kills,
            "jammer_active": snapshot.jammer_active,
            "events": [{"type": e.type, "msg": e.msg} for e in snapshot.events],
            "self": self._player_to_dict(snapshot.self_data) if snapshot.HasField("self_data") else None,
            "vision": {
                "players": [self._player_to_dict(p) for p in snapshot.visible_players],
                "entities": [self._entity_to_dict(e) for e in snapshot.visible_entities]
            },
            "radar_blips": [{"type": b.type, "pos": {"x": b.pos.x, "y": b.pos.y}} for b in snapshot.radar_blips],
            "sound": {
                "events": []  # TODO: 添加音效事件支持
            }
        }

    def _player_to_dict(self, player):
        """将 Player protobuf 转换为字典"""
        return {
            "session_id": player.session_id,
            "name": player.name,
            "pos": {"x": player.pos.x, "y": player.pos.y},
            "look_dir": {"x": player.look_dir.x, "y": player.look_dir.y},
            "hp": player.hp,
            "max_hp": player.max_hp,
            "armor": player.armor,
            "max_armor": player.max_armor,
            "is_alive": player.is_alive,
            "tactic": player.tactic,
            "inventory": [self._item_to_dict(item) for item in player.inventory],
            "money": player.money,
            "kills": player.kills,
            "is_extracted": player.is_extracted,
            "shop_stock": [self._item_to_dict(item) for item in player.shop_stock],
            # 可选字段
            "view_radius": player.view_radius if player.HasField("view_radius") else 8.0,
            "move_speed": player.move_speed if player.HasField("move_speed") else 5.0,
        }

    def _entity_to_dict(self, entity):
        """将 Entity protobuf 转换为字典"""
        return {
            "uid": entity.uid,
            "type": entity.type,
            "pos": {"x": entity.pos.x, "y": entity.pos.y},
            "extra": {}  # TODO: 根据需要解析 extra 字段
        }

    def _item_to_dict(self, item):
        """将 Item protobuf 转换为字典"""
        if not item or not item.id:
            return None
        return {
            "id": item.id,
            "name": item.name,
            "desc": item.desc,
            "price": item.price,
            "tier": item.tier
        }

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
