"""
MUD Bridge - persistent telnet connection controlled via files.

Connects to dogmud.org:55555 (AI port), logs in, then polls mud_cmd.txt
for commands and writes responses to mud_output.txt. A full session log
is appended to mud_log.txt.

Run in background; control by writing commands to mud_cmd.txt.
"""
import io
import os
import re
import socket
import sys
import time

# Force UTF-8 stdout on Windows to handle MUD Unicode chars
if sys.platform == "win32":
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
CMD_FILE = os.path.join(SCRIPT_DIR, "mud_cmd.txt")
OUT_FILE = os.path.join(SCRIPT_DIR, "mud_output.txt")
LOG_FILE = os.path.join(SCRIPT_DIR, "mud_log.txt")

MUD_HOST = os.environ.get("MUD_HOST", "dogmud.org")
MUD_PORT = int(os.environ.get("MUD_PORT", "55555"))
USERNAME = os.environ.get("AI_USERNAME", "aitester")
PASSWORD = os.environ.get("AI_PASSWORD", "testpass123")


def clean_text(text: str) -> str:
    text = re.sub(r'\x1b\[[0-9;]*[a-zA-Z]', '', text)
    text = text.replace('\r', '')
    text = re.sub(r'\n{3,}', '\n\n', text)
    return text.strip()


def recv_all(sock: socket.socket, timeout: float = 2.5) -> str:
    """Read everything available from the socket until a pause."""
    sock.setblocking(False)
    chunks = []
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            data = sock.recv(8192)
            if not data:
                break
            chunks.append(data.decode('utf-8', errors='replace'))
            time.sleep(0.1)  # brief pause to let more data arrive
        except BlockingIOError:
            if chunks:
                # Had data, now paused — probably done
                time.sleep(0.3)
                try:
                    data = sock.recv(8192)
                    if data:
                        chunks.append(data.decode('utf-8', errors='replace'))
                        continue
                except BlockingIOError:
                    pass
                break
            else:
                time.sleep(0.2)
    sock.setblocking(True)
    return "".join(chunks)


def send_cmd(sock: socket.socket, cmd: str) -> None:
    sock.sendall((cmd + "\r\n").encode('utf-8'))


def log(logfile, label: str, text: str) -> None:
    ts = time.strftime("%H:%M:%S")
    logfile.write(f"\n[{ts}] {label}\n{text}\n")
    logfile.flush()


def main():
    print(f"Connecting to {MUD_HOST}:{MUD_PORT}...")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(15)
    sock.connect((MUD_HOST, MUD_PORT))
    print("Connected.")

    logfile = open(LOG_FILE, "w", encoding="utf-8")

    # Read splash
    splash = clean_text(recv_all(sock, 10.0))
    log(logfile, "SPLASH", splash)
    print(splash[:500])

    def step(label, cmd, wait=2.0, read_wait=4.0):
        """Send a command, wait, read response, log and return it."""
        send_cmd(sock, cmd)
        time.sleep(wait)
        r = clean_text(recv_all(sock, read_wait))
        log(logfile, label, r)
        print(f"[{label}] {r[:200]}")
        return r

    def attempt_login():
        """Try to log in. Returns (success, final_resp)."""
        r = step("LOGIN_USER", USERNAME)
        if "password" not in r.lower():
            print(f"Unexpected response after username: {r[:200]}")
            return False, r
        r2 = step("LOGIN_PASS", PASSWORD, wait=3.0)
        if any(kw in r2.lower() for kw in ("incorrect", "invalid", "nope", "bye", "okay, bye")):
            return False, r2
        # Handle "already connected" kick prompt
        if "kick" in r2.lower() or "already connected" in r2.lower():
            r2 = step("LOGIN_KICK", "y", wait=3.0, read_wait=5.0)
        return True, r2

    def attempt_create():
        """Create a new account. Returns final response."""
        nonlocal sock
        sock.close()
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(15)
        sock.connect((MUD_HOST, MUD_PORT))

        splash2 = clean_text(recv_all(sock, 10.0))
        log(logfile, "SPLASH2", splash2)

        # New account creation flow:
        step("CREATE_NEW", "new")           # username prompt -> "new"
        step("CREATE_USERNAME", USERNAME)   # username-new prompt
        step("CREATE_PASS", PASSWORD)       # password-new prompt
        step("CREATE_PASS2", PASSWORD)      # password-new-verify
        step("CREATE_EMAIL", "")            # email (optional, skip)
        step("CREATE_SCREENREADER", "n")    # screen-reader
        r = step("CREATE_CONFIRM", "y", wait=3.0, read_wait=5.0)  # confirm
        return r

    success, resp = attempt_login()

    if not success:
        print("Login failed — trying account creation...")
        resp = attempt_create()
        if any(kw in resp.lower() for kw in ("okay, bye", "failed", "sorry", "error")):
            print(f"Account creation failed: {resp[:300]}")
            sock.close()
            sys.exit(1)
        # The MUD logs you in automatically after account creation —
        # check if resp already contains game content (HP bars, room text)
        if "HP:" in resp or "SP:" in resp or "You are" in resp or "☾" in resp:
            print("Account created and logged in automatically.")
        else:
            print("Account created! Logging in...")
            # Now log in with the newly created account
            time.sleep(2)
            success2, resp = attempt_login()
            if not success2:
                print(f"Login after creation failed: {resp[:300]}")
                sock.close()
                sys.exit(1)

    print("\nLogged in. Polling for commands...\n")

    # Initialize files
    with open(OUT_FILE, "w", encoding="utf-8") as f:
        f.write(resp)
    # Create empty cmd file
    open(CMD_FILE, "w").close()

    last_mtime = 0.0

    while True:
        try:
            time.sleep(0.5)

            # Check for unsolicited MUD output (combat ticks, regen, etc.)
            sock.setblocking(False)
            try:
                bg = sock.recv(8192)
                if bg:
                    bg_text = clean_text(bg.decode('utf-8', errors='replace'))
                    if bg_text:
                        log(logfile, "BG_OUTPUT", bg_text)
                        # Append to output file
                        with open(OUT_FILE, "a", encoding="utf-8") as f:
                            f.write("\n" + bg_text)
                elif bg == b"":
                    # Connection closed by server
                    log(logfile, "BG_ERROR", "Server closed connection (recv returned empty bytes)")
                    break
            except BlockingIOError:
                pass
            except OSError as _e:
                # Non-blocking would-block errors are fine; others indicate disconnect
                import errno as _errno
                if hasattr(_e, 'errno') and _e.errno in (
                    _errno.EAGAIN, getattr(_errno, 'EWOULDBLOCK', 11),
                    getattr(_errno, 'WSAEWOULDBLOCK', 10035)
                ):
                    pass
                else:
                    log(logfile, "BG_ERROR", f"OSError in bg recv: {_e}")
                    break
            finally:
                sock.setblocking(True)

            # Check for new command
            try:
                mtime = os.path.getmtime(CMD_FILE)
            except FileNotFoundError:
                continue

            if mtime > last_mtime:
                last_mtime = mtime
                with open(CMD_FILE, "r", encoding="utf-8") as f:
                    cmd = f.read().strip()
                if not cmd:
                    continue

                log(logfile, "SEND_CMD", cmd)
                print(f"> {cmd}")
                send_cmd(sock, cmd)

                # Wait for response
                time.sleep(2.5)
                resp = clean_text(recv_all(sock, 3.0))

                log(logfile, "RESPONSE", resp)
                with open(OUT_FILE, "w", encoding="utf-8") as f:
                    f.write(resp)

                # Print truncated to console
                display = resp[:400]
                if len(resp) > 400:
                    display += f"\n... ({len(resp)} chars)"
                print(display)
                print()

        except KeyboardInterrupt:
            print("\nDisconnecting...")
            try:
                send_cmd(sock, "quit")
                time.sleep(1)
            except Exception:
                pass
            break
        except (ConnectionError, OSError) as e:
            log(logfile, "CONN_ERROR", str(e))
            print(f"Connection lost: {e}")
            break
        except Exception as e:
            import traceback
            tb = traceback.format_exc()
            log(logfile, "FATAL_ERROR", tb)
            print(f"Unexpected error: {e}\n{tb}")
            break

    sock.close()
    logfile.close()
    print("Bridge closed.")


if __name__ == "__main__":
    main()
