import argparse
import json
import sys
import time

from websocket import WebSocket, WebSocketTimeoutException


def main() -> int:
    p = argparse.ArgumentParser(description="Echo_Trace WebSocket connectivity test")
    p.add_argument(
        "--url",
        default="ws://139.224.0.73:9191/ws",
        help="WebSocket URL, e.g. ws://host:9191/ws",
    )
    p.add_argument("--timeout", type=float, default=5.0, help="Connect/recv timeout seconds")
    p.add_argument(
        "--send",
        default="",
        help=(
            "Optional JSON string to send after connecting. "
            "Example: '{\"type\":1001,\"payload\":{}}'"
        ),
    )
    p.add_argument(
        "--recv",
        type=int,
        default=1,
        help="How many messages to try receiving after connect/send (0 to skip)",
    )

    args = p.parse_args()

    ws = WebSocket()
    ws.settimeout(args.timeout)

    t0 = time.time()
    try:
        ws.connect(args.url)
    except Exception as e:
        dt = time.time() - t0
        print(f"CONNECT FAILED after {dt:.2f}s: {e}")
        return 1

    dt = time.time() - t0
    print(f"CONNECTED in {dt:.2f}s -> {args.url}")

    if args.send:
        try:
            payload = json.loads(args.send)
        except Exception as e:
            print(f"Invalid --send JSON: {e}")
            ws.close()
            return 2

        msg = json.dumps(payload, ensure_ascii=False)
        print(f"SEND: {msg}")
        ws.send(msg)

    for i in range(max(0, int(args.recv))):
        try:
            msg = ws.recv()
        except WebSocketTimeoutException:
            print("RECV TIMEOUT")
            break
        except Exception as e:
            print(f"RECV ERROR: {e}")
            break

        print(f"RECV[{i}]: {msg}")

    ws.close()
    print("CLOSED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
