import argparse
import sys
import time

from client_base import GameClient


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9000)
    parser.add_argument("--mode", choices=["tcp", "ws"], default="tcp")
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument("--ws-use-json", action="store_true")
    args = parser.parse_args()

    server_addr = (args.host, args.port)
    client = GameClient(server_addr, mode=args.mode, ws_path=args.ws_path, ws_use_json=args.ws_use_json)
    client.connect()
    client.login(account_id="test1", token="test1")

    while True:
        cmd = input("> ").strip()
        if cmd == "quit":
            sys.exit(0)
        if cmd == "close":
            client.close()
        elif cmd == "reconnect":
            client.close()
            time.sleep(1)
            client.connect()
            client.resume()
        elif cmd == "login":
            client.login(account_id="test1", token="test1")
        elif cmd == "load":
            client.load_player_data()
        else:
            print("commands: login | load | close | reconnect | quit")


if __name__ == "__main__":
    main()
