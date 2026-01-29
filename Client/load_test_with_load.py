import argparse
import random
import statistics
import threading
import time

from client_base import (
    GameClient,
    MSG_LOAD_PLAYER_DATA_RSP,
    MSG_LOGIN_RSP,
)

def run_pressure_client(server_addr, idx, mode, ws_path, ws_use_json, rounds=20, interval=0.1):
    client = GameClient(server_addr, mode=mode, ws_path=ws_path, ws_use_json=ws_use_json)
    client.connect()

    account = f"test{idx}"
    client.login(token=account, account_id=account)

    if not client.login_event.wait(timeout=5.0):
        print(f"[Client {idx}] login timeout")
        client.close()
        return []

    latencies = []

    for i in range(rounds):
        client.load_event.clear()
        start = time.perf_counter()

        client.load_player_data()
        if not client.load_event.wait(timeout=5.0):
            print(f"[Client {idx}] load timeout")
            break

        latencies.append(time.perf_counter() - start)
        time.sleep(interval)

    # 模拟在线停留
    time.sleep(random.uniform(10, 20))
    client.close()
    return latencies

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9000)
    parser.add_argument("--mode", choices=["tcp", "ws"], default="tcp")
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument("--ws-use-json", action="store_true")
    args = parser.parse_args()

    server_addr = (args.host, args.port)

    threads = []
    all_latencies = []
    lock = threading.Lock()

    def worker(idx):
        lats = run_pressure_client(
            server_addr,
            idx,
            args.mode,
            args.ws_path,
            args.ws_use_json,
            rounds=1,
            interval=0.1,
        )
        with lock:
            all_latencies.extend(lats)

    for i in range(1000):
        t = threading.Thread(target=worker, args=(i,))
        t.start()
        threads.append(t)
        time.sleep(0.02)

    for t in threads:
        t.join()

    if all_latencies:
        all_latencies.sort()
        print("count:", len(all_latencies))
        print("p50:", all_latencies[int(len(all_latencies) * 0.5)])
        print("p95:", all_latencies[int(len(all_latencies) * 0.95)])
        print("p99:", all_latencies[int(len(all_latencies) * 0.99)])

if __name__ == "__main__":
    main()
