import asyncio
import struct
import time
import random
import statistics

from internal_pb.internal_pb2 import Envelope
from internal_pb.gate_pb2 import ResumeReq, SessionInit
from internal_pb.login_pb2 import LoginReq, LoginRsp
from internal_pb.game_pb2 import LoadPlayerDataReq, LoadPlayerDataRsp, PlayerInitRsp


# =====================
# Msg IDs
# =====================
MSG_RESUME_REQ = 1
MSG_RESUME_RSP = 2
MSG_SESSION_INIT = 3

MSG_HEARTBEAT_REQ = 10
MSG_HEARTBEAT_RSP = 11

MSG_LOGIN_REQ = 1001
MSG_LOGIN_RSP = 1002

MSG_ENTER_GAME_RSP = 3002
MSG_LOAD_PLAYER_DATA_REQ = 3003
MSG_LOAD_PLAYER_DATA_RSP = 3004
MSG_PLAYER_OFFLINE_NOTIFY = 3006


# =====================
# Async Game Client
# =====================
class AsyncGameClient:
    def __init__(self, host, port, idx):
        self.host = host
        self.port = port
        self.idx = idx

        self.reader: asyncio.StreamReader = None
        self.writer: asyncio.StreamWriter = None

        self.session_id = 0
        self.player_id = 0
        self.token = ""

        self.running = True

        # events
        self.login_event = asyncio.Event()
        self.load_event = asyncio.Event()

        # heartbeat
        self.heartbeat_task = None

    # -----------------
    async def connect(self):
        self.reader, self.writer = await asyncio.open_connection(
            self.host, self.port
        )
        asyncio.create_task(self.recv_loop())

    async def close(self):
        self.running = False
        if self.writer:
            self.writer.close()
            try:
                await self.writer.wait_closed()
            except Exception:
                pass

    # -----------------
    async def send_envelope(self, msg_id, payload=b""):
        if not self.running:
            return

        env = Envelope(
            msg_id=msg_id,
            session_id=self.session_id,
            player_id=self.player_id,
            payload=payload,
        )
        data = env.SerializeToString()
        self.writer.write(struct.pack(">I", len(data)) + data)
        await self.writer.drain()

    # -----------------
    async def login(self):
        account = f"test{self.idx}"
        req = LoginReq(
            token=account,
            account_id=account,
            platform=0,
        )
        await self.send_envelope(MSG_LOGIN_REQ, req.SerializeToString())

    async def load_player_data(self):
        req = LoadPlayerDataReq()
        await self.send_envelope(
            MSG_LOAD_PLAYER_DATA_REQ, req.SerializeToString()
        )

    # -----------------
    async def start_heartbeat(self):
        if self.heartbeat_task:
            return
        self.heartbeat_task = asyncio.create_task(self.heartbeat_loop())

    async def heartbeat_loop(self):
        while self.running:
            await asyncio.sleep(5)
            try:
                await self.send_envelope(MSG_HEARTBEAT_REQ)
            except Exception:
                return

    # -----------------
    async def recv_loop(self):
        try:
            while self.running:
                header = await self.reader.readexactly(4)
                size = struct.unpack(">I", header)[0]
                body = await self.reader.readexactly(size)

                env = Envelope()
                env.ParseFromString(body)
                await self.on_message(env)

        except (asyncio.IncompleteReadError, ConnectionResetError):
            pass
        finally:
            await self.close()

    # -----------------
    async def on_message(self, env: Envelope):
        if env.msg_id == MSG_SESSION_INIT:
            init = SessionInit()
            init.ParseFromString(env.payload)
            self.session_id = init.session_id
            self.token = init.token

        elif env.msg_id == MSG_LOGIN_RSP:
            rsp = LoginRsp()
            rsp.ParseFromString(env.payload)
            self.player_id = rsp.player_id
            self.login_event.set()
            await self.start_heartbeat()

        elif env.msg_id == MSG_ENTER_GAME_RSP:
            pass

        elif env.msg_id == MSG_RESUME_RSP:
            await self.start_heartbeat()

        elif env.msg_id == MSG_LOAD_PLAYER_DATA_RSP:
            self.load_event.set()

        elif env.msg_id == MSG_HEARTBEAT_RSP:
            pass

        elif env.msg_id == MSG_PLAYER_OFFLINE_NOTIFY:
            pass


# =====================
# Pressure Worker
# =====================
async def run_pressure_client(
    host, port, idx, rounds=10, interval=0.1
):
    client = AsyncGameClient(host, port, idx)
    await client.connect()
    await client.login()

    try:
        await asyncio.wait_for(client.login_event.wait(), timeout=5.0)
    except asyncio.TimeoutError:
        print("======222222222=========")
        await client.close()
        return []

    latencies = []

    for _ in range(rounds):
        client.load_event.clear()
        start = time.perf_counter()

        await client.load_player_data()
        try:
            await asyncio.wait_for(client.load_event.wait(), timeout=300.0)
        except asyncio.TimeoutError:
            print("======111111=========")
            break

        latencies.append(time.perf_counter() - start)
        await asyncio.sleep(interval)

    # 模拟在线停留
    await asyncio.sleep(random.uniform(1, 2))
    await client.close()
    return latencies


# =====================
# Main
# =====================
async def main():
    host = "127.0.0.1"
    port = 9000

    client_count = 500
    rounds = 1

    tasks = []

    for i in range(client_count):
        task = asyncio.create_task(
            run_pressure_client(host, port, i, rounds=rounds)
        )
        tasks.append(task)

        # ⭐ 核心：限制建连速率（1~5ms 都行）
        await asyncio.sleep(0.002)  # 2ms 一个连接

    results = await asyncio.gather(*tasks)

    all_latencies = [lat for sub in results for lat in sub]
    if not all_latencies:
        print("no success samples")
        return

    all_latencies.sort()

    def pct(p):
        return all_latencies[int(len(all_latencies) * p)]

    print("clients:", client_count)
    print("samples:", len(all_latencies))
    print("p50:", pct(0.5))
    print("p95:", pct(0.95))
    print("p99:", pct(0.99))


if __name__ == "__main__":
    asyncio.run(main())
