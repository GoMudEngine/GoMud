# DOGMud - Delusions of Grandeur

**Delusions of Grandeur** (DOG) is a MUD set on Gaius, a colony world where a crashed ship's survivors have forgotten their origins. A symbiotic organism called The Chrysalis infects the population, making beliefs and convictions become reality -- manifesting as "magic" and physical mutations.

Built on the [GoMud](https://github.com/GoMudEngine/GoMud) engine.

## World Design

- **Setting**: Planet Gaius, continent of Thera, in the Windward Marches -- a region inspired by the Pacific Northwest with diverse microclimates
- **Species**: All players are human; non-human species exist as NPCs and creatures
- **Progression**: Skill-based (no levels or XP) -- skills and stats improve through use, critical successes/failures, and notable deeds
- **Combat**: Distribution-based rolling instead of traditional dice, with stamina management adding tactical depth
- **Magic**: Powered by Conviction (belief made real through The Chrysalis infection) rather than mana
- **Mutations**: The Chrysalis grants physical mutations and reality-altering powers based on a character's beliefs
- **Stats**: Strength, Dexterity, Perception, Vitality, Willpower, Charisma
- **Resource Pools**: Health, Stamina, Conviction
- **Three Moons**: Swiftmoon, The Wanderer, and The Eye orbit Gaius, affecting stats based on their phases

See [world.md](world.md) for the full design document.

## Current Development Status

See [DEVELOPMENT_PLAN.md](DEVELOPMENT_PLAN.md) for the implementation roadmap. Active work is organized into incremental stages that keep the MUD playable at every step.

## Connecting

_TELNET_ : connect to `localhost` on port `33333` with a telnet client

_WEB CLIENT_: [http://localhost/webclient](http://localhost/webclient)

**Default Username:** _admin_

**Default Password:** _password_

## Env Vars

When running several environment variables can be set to alter behaviors of the mud:

- **CONFIG_PATH**_=/path/to/alternative/config.yaml_ - This can provide a path to a copy of the config.yaml containing only values you wish to override. This way you don't have to modify the original config.yaml
- **LOG_PATH**_=/path/to/log.txt_ - This will write all logs to a specified file. If unspecified, will write to _stderr_.
- **LOG_LEVEL**_={LOW/MEDIUM/HIGH}_ - This sets how verbose you want the logs to be. _(Note: Log files rotate every 100MB)_
- **LOG_NOCOLOR**_=1_ - If set, logs will be written without colorization.

---

## Built on GoMud

DOGMud is built on [GoMud](https://github.com/GoMudEngine/GoMud), an open source MUD engine written in Go by [the GoMud team](https://github.com/GoMudEngine).

GoMud provides the networking, templating, scripting, and world engine that DOGMud builds upon. If you're interested in building your own MUD, check out the original project:

- [GoMud GitHub](https://github.com/GoMudEngine/GoMud)
- [GoMud Discord](https://discord.gg/cjukKvQWyy)
- [GoMud Discussions](https://github.com/GoMudEngine/GoMud/discussions)
- [GoMud Guides](_datafiles/guides/README.md)
- [GoMud Contributing Guide](https://github.com/GoMudEngine/GoMud/blob/master/.github/CONTRIBUTING.md)

### GoMud Feature Demos

- [Auto-complete input](https://youtu.be/7sG-FFHdhtI)
- [In-game maps](https://youtu.be/navCCH-mz_8)
- [Quests / Quest Progress](https://youtu.be/3zIClk3ewTU)
- [Lockpicking](https://youtu.be/-zgw99oI0XY)
- [Hired Mercs](https://youtu.be/semi97yokZE)
- [TinyMap](https://www.youtube.com/watch?v=VLNF5oM4pWw)
- [256 Color/xterm](https://www.youtube.com/watch?v=gGSrLwdVZZQ)
- [Customizable Prompts](https://www.youtube.com/watch?v=MFkmjSTL0Ds)
- [Mob/NPC Scripting](https://www.youtube.com/watch?v=li2k1N4p74o)
- [Room Scripting](https://www.youtube.com/watch?v=n1qNUjhyOqg)
- [Kill Stats](https://www.youtube.com/watch?v=4aXs8JNj5Cc)
- [Searchable Inventory](https://www.youtube.com/watch?v=iDUbdeR2BUg)
- [Day/Night Cycles](https://www.youtube.com/watch?v=CiEbOp244cw)
- [Web Socket "Virtual Terminal"](https://www.youtube.com/watch?v=L-qtybXO4aw)
- [Alternate Characters](https://www.youtube.com/watch?v=VERF2l70W34)

### ANSI Colors

Colorization is handled through extensive use of [github.com/GoMudEngine/ansitags](https://github.com/GoMudEngine/ansitags).

### Why Go?

Why not?

Go provides a lot of terrific benefits such as:

- Compatible - High degree of compatibility across platforms or CPU Architectures.
- Fast - Go is fast. From execution to builds.
- Opinionated - Go style and patterns are well established.
- Modern - A relatively new/modern language without decades of accumulated complexity.
- Upgradable - Go's backward compatibility promise makes upgrades painless.
- Statically Linked - If you have the binary, you have the working program.
- No Central Registries - Library includes straight from their repos.
- Concurrent - Concurrency built in as a language feature, not a library.
