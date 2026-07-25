"""Strip the inert `cooldown:` key from room spawninfo blocks.

`cooldown` is not a field on rooms.SpawnInfo -- the real field is
`respawnrate` -- so these authored lines have never done anything. The boot
smoke test has carried them as a known silently-ignored key with the note
"authored values doing nothing".

They are DELETED rather than converted. The values are in ROUNDS while every
live respawnrate in the world is in real minutes, and at RoundSeconds 4 a
verbatim conversion would make these starter-area spawn points 1.3-11x slower
to return -- a gameplay change nobody asked for, arriving as a side effect of
a cleanup. These spawns keep the 15-minute default they have always had. The
authored values are recorded in the spec for a future deliberate pacing pass.

Line-level edit, not a YAML round-trip: re-marshalling reflows every
description block in the file, turning a 36-line deletion into a diff across
the whole world.
"""

import glob
import io
import re

changed, removed = 0, 0
for path in glob.glob('_datafiles/world/dogmud/rooms/*/[0-9]*.yaml'):
    src = io.open(path, encoding='utf-8', newline='').read()
    eol = '\r\n' if '\r\n' in src else '\n'
    lines = src.split(eol)
    # Indentation-agnostic on purpose. Spawn lists are authored both flush
    # ("- mobid:", keys at 2 spaces) and indented ("  - mobid:", keys at 4),
    # and an anchor on one depth silently misses the other -- which is exactly
    # what happened on the first pass here. `cooldown` is not a valid key
    # anywhere on rooms.Room, so any indented occurrence in a room file is the
    # same inert key.
    keep = [ln for ln in lines if not re.match(r'^\s+cooldown:', ln)]
    if len(keep) != len(lines):
        removed += len(lines) - len(keep)
        changed += 1
        io.open(path, 'w', encoding='utf-8', newline='').write(eol.join(keep))

print('files changed: %d   cooldown lines removed: %d' % (changed, removed))
