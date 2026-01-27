import socket
import struct
import threading
import time

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


def make_resume_token(session_id: int) -> str:
    return f"session:{session_id}"


class GameClient:
    def __init__(self, server_addr):
        self.server_addr = server_addr
        self.sock = None
        self.session_id = 0
        self.player_id = 0
        self.token = ""
        self.running = False
        self.lock = threading.Lock()

    def connect(self):
        self.sock = socket.socket()
        self.sock.connect(self.server_addr)
        self.running = True
        threading.Thread(target=self.recv_loop, daemon=True).start()
        print("[Client] connected")

    def close(self):
        self.running = False
        if self.sock:
            self.sock.close()
            self.sock = None
        print("[Client] disconnected")

    def send_envelope(self, msg_id, payload=b""):
        with self.lock:
            env = Envelope(
                msg_id=msg_id,
                session_id=self.session_id,
                player_id = self.player_id,
                payload=payload,
            )
            data = env.SerializeToString()
            pkt = struct.pack(">I", len(data)) + data
            self.sock.sendall(pkt)

    def login(self, token="test-token", account_id="test1", platform=0):
        req = LoginReq(token=token, account_id=account_id, platform=platform)
        print("[Client] send LoginReq")
        self.send_envelope(MSG_LOGIN_REQ, req.SerializeToString())

    def resume(self):
        if not self.session_id:
            raise RuntimeError("session_id not set, cannot resume")
        req = ResumeReq(session_id=self.session_id, token=self.token)
        print("[Client] send ResumeReq")
        self.send_envelope(MSG_RESUME_REQ, req.SerializeToString())

    def load_player_data(self):
        req = LoadPlayerDataReq()
        print("[Client] send LoadPlayerDataReq")
        self.send_envelope(MSG_LOAD_PLAYER_DATA_REQ, req.SerializeToString())

    def heartbeat_loop(self):
        while self.running:
            time.sleep(5)
            try:
                self.send_envelope(MSG_HEARTBEAT_REQ)
                print("[Client] heartbeat")
            except Exception as exc:
                print("[Client] heartbeat failed:", exc)
                return

    def recv_loop(self):
        try:
            while self.running:
                header = self._recv_exact(4)
                if not header:
                    break
                size = struct.unpack(">I", header)[0]
                body = self._recv_exact(size)
                env = Envelope()
                env.ParseFromString(body)
                self.on_message(env)
        except Exception as exc:
            print("[Client] recv error:", exc)
        self.close()

    def _recv_exact(self, n):
        data = b""
        while len(data) < n:
            chunk = self.sock.recv(n - len(data))
            if not chunk:
                return None
            data += chunk
        return data

    def on_message(self, env: Envelope):
        if env.msg_id == MSG_SESSION_INIT:
            print(f"[Client] receive session_id = {env.session_id}")
            init = SessionInit()
            init.ParseFromString(env.payload)
            self.session_id = env.session_id
            self.token = init.token

        if env.msg_id == MSG_LOGIN_RSP:
            rsp = LoginRsp()
            rsp.ParseFromString(env.payload)
            print(f"[Client] LoginRsp player={rsp.player_id}")
            self.player_id=rsp.player_id
            threading.Thread(target=self.heartbeat_loop, daemon=True).start()
        elif env.msg_id == MSG_ENTER_GAME_RSP:
            rsp = PlayerInitRsp()
            rsp.ParseFromString(env.payload)
            print(f"[Client] EnterGameRsp role={rsp.data.role_id}")
        elif env.msg_id == MSG_RESUME_RSP:
            print("[Client] ResumeRsp OK")
            threading.Thread(target=self.heartbeat_loop, daemon=True).start()
        elif env.msg_id == MSG_HEARTBEAT_RSP:
            print("[Client] heartbeat rsp")
        elif env.msg_id == MSG_LOAD_PLAYER_DATA_RSP:
            rsp = LoadPlayerDataRsp()
            rsp.ParseFromString(env.payload)
            print(f"[Client] LoadPlayerDataRsp role={rsp.data.role_id}")
        else:
            print("[Client] recv msg:", env.msg_id)
