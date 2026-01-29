import socket
import struct
import threading
import time
import websocket

from google.protobuf import json_format
from internal_pb.internal_pb2 import Envelope
from internal_pb.gate_pb2 import ResumeReq, SessionInit
from internal_pb.login_pb2 import LoginReq, LoginRsp
from internal_pb.game_pb2 import LoadPlayerDataReq, LoadPlayerDataRsp, PlayerInitRsp

MSG_RESUME_REQ = 1
MSG_RESUME_RSP = 2
MSG_SESSION_INIT = 3

MSG_HEARTBEAT_REQ = 10
MSG_HEARTBEAT_RSP = 11
MSG_LOGIN_REQ = 1001
MSG_LOGIN_RSP = 1002
MSG_ENTER_GAME_REQ = 3001
MSG_ENTER_GAME_RSP = 3002
MSG_LOAD_PLAYER_DATA_REQ = 3003
MSG_LOAD_PLAYER_DATA_RSP = 3004
MSG_PLAYER_OFFLINE_NOTIFY = 3006

# =====================
# Client
# =====================
class GameClient:
    def __init__(self, server_addr, mode="tcp", ws_path="/ws", ws_use_json=False):
        self.server_addr = server_addr
        self.mode = mode
        self.ws_path = ws_path
        self.ws_use_json = ws_use_json

        self.sock = None
        self.ws = None
        self.running = False

        self.session_id = 0
        self.player_id = 0
        self.token = ""

        self.lock = threading.Lock()

        # thread control
        self.recv_thread = None
        self.heartbeat_thread = None
        self.heartbeat_started = False

        # events
        self.login_event = threading.Event()
        self.load_event = threading.Event()

    # -----------------
    # Network
    # -----------------
    def connect(self):
        if self.mode == "tcp":
            self.sock = socket.socket()
            self.sock.connect(self.server_addr)
        else:
            self.ws = websocket.create_connection(self._ws_url())
        self.running = True

        self.recv_thread = threading.Thread(
            target=self.recv_loop, daemon=True
        )
        self.recv_thread.start()

        print("[Client] connected")

    def close(self):
        if not self.running:
            return

        self.running = False

        if self.sock:
            try:
                self.sock.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            try:
                self.sock.close()
            except OSError:
                pass
            self.sock = None
        if self.ws:
            try:
                self.ws.close()
            except OSError:
                pass
            self.ws = None

        print("[Client] disconnected")

    # -----------------
    # Send
    # -----------------
    def send_envelope(self, msg_id, payload=b""):
        if not self.running:
            return

        with self.lock:
            env = Envelope(
                msg_id=msg_id,
                session_id=self.session_id,
                player_id=self.player_id,
                payload=payload,
            )
            if self.mode == "tcp":
                data = env.SerializeToString()
                pkt = struct.pack(">I", len(data)) + data
                self.sock.sendall(pkt)
            else:
                if self.ws_use_json:
                    data = json_format.MessageToJson(env)
                    self.ws.send(data)
                else:
                    data = env.SerializeToString()
                    self.ws.send(data, opcode=websocket.ABNF.OPCODE_BINARY)

    # -----------------
    # API
    # -----------------
    def login(self, token, account_id, platform=0):
        req = LoginReq(
            token=token,
            account_id=account_id,
            platform=platform,
        )
        print("[Client] send LoginReq")
        self.send_envelope(MSG_LOGIN_REQ, req.SerializeToString())

    def resume(self):
        if not self.session_id:
            return
        req = ResumeReq(
            session_id=self.session_id,
            token=self.token,
        )
        print("[Client] send ResumeReq")
        self.send_envelope(MSG_RESUME_REQ, req.SerializeToString())

    def load_player_data(self):
        req = LoadPlayerDataReq()
        self.send_envelope(MSG_LOAD_PLAYER_DATA_REQ, req.SerializeToString())

    # -----------------
    # Heartbeat（关键）
    # -----------------
    def start_heartbeat(self):
        if self.heartbeat_started:
            return

        self.heartbeat_started = True
        self.heartbeat_thread = threading.Thread(
            target=self.heartbeat_loop, daemon=True
        )
        self.heartbeat_thread.start()

    def heartbeat_loop(self):
        """
        ✔ 不抛异常
        ✔ 断线自动退出
        ✔ resume 后继续可用
        """
        while self.running:
            time.sleep(5)

            if not self.running or not self.sock:
                return

            try:
                self.send_envelope(MSG_HEARTBEAT_REQ)
            except OSError:
                # socket 已关闭，静默退出
                return

    # -----------------
    # Recv
    # -----------------
    def recv_loop(self):
        if self.mode == "tcp":
            self._recv_loop_tcp()
        else:
            self._recv_loop_ws()

    def _recv_loop_tcp(self):
        try:
            while self.running:
                header = self._recv_exact(4)
                if not header:
                    break

                size = struct.unpack(">I", header)[0]
                body = self._recv_exact(size)
                if not body:
                    break

                env = Envelope()
                env.ParseFromString(body)
                self.on_message(env)

        except OSError:
            # 主动 close / 网络中断，正常情况
            pass
        finally:
            self.close()

    def _recv_loop_ws(self):
        try:
            while self.running:
                data = self.ws.recv()
                if not data:
                    break
                env = Envelope()
                if self.ws_use_json:
                    json_format.Parse(data, env)
                else:
                    env.ParseFromString(data)
                self.on_message(env)
        except OSError:
            pass
        finally:
            self.close()

    def _recv_exact(self, n):
        data = b""
        while len(data) < n and self.running:
            chunk = self.sock.recv(n - len(data))
            if not chunk:
                return None
            data += chunk
        return data

    def _ws_url(self):
        if isinstance(self.server_addr, str):
            if self.server_addr.startswith("ws://") or self.server_addr.startswith("wss://"):
                return self.server_addr
            return f"ws://{self.server_addr}{self.ws_path}"
        host, port = self.server_addr
        return f"ws://{host}:{port}{self.ws_path}"

    # -----------------
    # Message
    # -----------------
    def on_message(self, env: Envelope):
        if env.msg_id == MSG_SESSION_INIT:
            init = SessionInit()
            init.ParseFromString(env.payload)
            self.session_id = init.session_id
            self.token = init.token
            print(f"[Client] SessionInit session={self.session_id}")

        elif env.msg_id == MSG_LOGIN_RSP:
            rsp = LoginRsp()
            rsp.ParseFromString(env.payload)
            self.player_id = rsp.player_id
            print(f"[Client] LoginRsp player={self.player_id}")
            self.login_event.set()
            self.start_heartbeat()

        elif env.msg_id == MSG_ENTER_GAME_RSP:
            rsp = PlayerInitRsp()
            rsp.ParseFromString(env.payload)
            print(f"[Client] EnterGameRsp role={rsp.data.role_id}")

        elif env.msg_id == MSG_RESUME_RSP:
            print("[Client] ResumeRsp OK")
            self.start_heartbeat()

        elif env.msg_id == MSG_HEARTBEAT_RSP:
            pass

        elif env.msg_id == MSG_LOAD_PLAYER_DATA_RSP:
            print(f"[Client] MSG_LOAD_PLAYER_DATA_RSP role={env.player_id}")
            rsp = LoadPlayerDataRsp()
            rsp.ParseFromString(env.payload)
            self.load_event.set()

        elif env.msg_id == MSG_PLAYER_OFFLINE_NOTIFY:
            print(f"[Client] PlayerOffline role={env.player_id}")

        else:
            print("[Client] recv msg:", env.msg_id)
