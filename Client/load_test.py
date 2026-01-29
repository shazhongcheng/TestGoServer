import argparse
import threading
import time

from client_base import GameClient


def run_client(server_addr, idx, mode, ws_path, ws_use_json):
    client = GameClient(server_addr, mode=mode, ws_path=ws_path, ws_use_json=ws_use_json)
    client.connect()
    account_id = f"test{idx}"
    client.login(token=account_id, account_id=account_id)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9000)
    parser.add_argument("--count", type=int, default=100)
    parser.add_argument("--delay", type=float, default=0.01)
    parser.add_argument("--mode", choices=["tcp", "ws"], default="tcp")
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument("--ws-use-json", action="store_true")
    args = parser.parse_args()

    server_addr = (args.host, args.port)
    threads = []
    for idx in range(1, args.count + 1):
        thread = threading.Thread(
            target=run_client,
            args=(server_addr, idx, args.mode, args.ws_path, args.ws_use_json),
            daemon=True,
        )
        thread.start()
        threads.append(thread)
        time.sleep(args.delay)

    for thread in threads:
        thread.join()


if __name__ == "__main__":
    main()
