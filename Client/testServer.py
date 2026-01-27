import socket
import struct
import threading
import time
import sys

from internal_pb.internal_pb2 import Envelope
from internal_pb.gate_pb2 import ResumeReq

SERVER_ADDR = ("127.0.0.1", 9000)

MSG_RESUME_REQ = 1
MSG_RESUME_RSP = 2
MSG_HEARTBEAT_REQ = 10
MSG_HEARTBEAT_RSP = 11


class Client:
    def __init__(self):
        self.sock = None
        self.session_id = 1       # 测试写死
        self.token = "xxx"        # 测试写死
        self.running = False

    # ---------- 网络基础 ----------
    def connect(self):
        self.sock = socket.socket()
        self.sock.connect(SERVER_ADDR)
        self.running = True
        print("[Client] connected")

        threading.Thread(target=self.recv_loop, daemon=True).start()

    def close(self):
        self.running = False
        if self.sock:
            self.sock.close()
            self.sock = None
        print("[Client] disconnected")

    def send_envelope(self, msg_id, payload=b""):
        env = Envelope(
            msg_id=msg_id,
            session_id=self.session_id,
            payload=payload
        )
        data = env.SerializeToString()
        pkt = struct.pack(">I", len(data)) + data
        self.sock.sendall(pkt)

    # ---------- 协议 ----------
    def resume(self):
        req = ResumeReq(
            session_id=self.session_id,
            token=self.token
        )
        print("[Client] send ResumeReq")
        self.send_envelope(MSG_RESUME_REQ, req.SerializeToString())

    def heartbeat_loop(self):
        while self.running:
            time.sleep(5)
            try:
                self.send_envelope(MSG_HEARTBEAT_REQ)
                print("[Client] heartbeat")
            except Exception as e:
                print("[Client] heartbeat failed:", e)
                return

    # ---------- 接收 ----------
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

        except Exception as e:
            print("[Client] recv error:", e)

        self.close()

    def _recv_exact(self, n):
        data = b""
        while len(data) < n:
            chunk = self.sock.recv(n - len(data))
            if not chunk:
                return None
            data += chunk
        return data

    def on_message(self, env):
        if env.msg_id == MSG_RESUME_RSP:
            print("[Client] ResumeRsp OK")
            threading.Thread(target=self.heartbeat_loop, daemon=True).start()
        elif env.msg_id == MSG_HEARTBEAT_RSP:
            print("[Client] heartbeat rsp")
        else:
            print("[Client] recv msg:", env.msg_id)


# ---------- 控制台 ----------
def main():
    c = Client()
    c.connect()
    c.resume()

    while True:
        cmd = input("> ").strip()
        if cmd == "quit":
            sys.exit(0)
        elif cmd == "close":
            c.close()
        elif cmd == "reconnect":
            c.close()
            time.sleep(1)
            c.connect()
            c.resume()
        else:
            print("commands: close | reconnect | quit")


if __name__ == "__main__":
    main()
