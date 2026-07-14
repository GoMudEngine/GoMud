#!/usr/bin/env python3
"""Probe a running MUD for its MSSP (telnet option 70) status block.

Usage: python tools/mssp_probe.py [host] [port]
Defaults: 127.0.0.1 33333
"""
import socket
import sys

host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
port = int(sys.argv[2]) if len(sys.argv) > 2 else 33333
IAC, SB, SE, DO, WILL, MSSP = 255, 250, 240, 253, 251, 70

s = socket.create_connection((host, port), timeout=10)
s.sendall(bytes([IAC, DO, MSSP]))  # ask for MSSP
s.settimeout(4)
buf = b""
try:
    while True:
        d = s.recv(4096)
        if not d:
            break
        buf += d
        start = buf.find(bytes([IAC, SB, MSSP]))
        if start >= 0 and buf.find(bytes([IAC, SE]), start) >= 0:
            break
except socket.timeout:
    pass
s.close()

start = buf.find(bytes([IAC, SB, MSSP]))
if start < 0:
    print("No MSSP reply received")
    sys.exit(1)
body = buf[start + 3:]
end = body.find(bytes([IAC, SE]))
if end >= 0:
    body = body[:end]

out, i = [], 0
while i < len(body):
    if body[i] == 1:  # MSSP_VAR
        i += 1
        name = b""
        while i < len(body) and body[i] not in (1, 2):
            name += bytes([body[i]])
            i += 1
        out.append("\n" + name.decode(errors="replace") + " =")
    elif body[i] == 2:  # MSSP_VAL
        i += 1
        val = b""
        while i < len(body) and body[i] not in (1, 2):
            val += bytes([body[i]])
            i += 1
        out.append(" " + val.decode(errors="replace"))
    else:
        i += 1
print("".join(out).strip())
