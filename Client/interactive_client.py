import sys
import time

from client_base import GameClient

SERVER_ADDR = ("127.0.0.1", 9000)


def main():
    client = GameClient(SERVER_ADDR)
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
