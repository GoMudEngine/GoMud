# Item Scripting

Example Script: 
* [Item Script](/_datafiles/world/default/items/other-0/6-sleeping_bag.js)

## Script paths

All item scripts reside the same folder as the item definition file.

For example, the item located at [/_datafiles/world/default/items/other-0/6-sleeping_bag.yaml](/_datafiles/world/default/items/other-0/6-sleeping_bag.yaml) would place its script at [/_datafiles/world/default/items/other-0/6-sleeping_bag.js](/_datafiles/world/default/items/other-0/6-sleeping_bag.js)

# Script Functions and Rules

The following functions are special keywords that will be invoked under specific circumstances if they are defined within your script:

---

```
function onLost(user ActorObject, item ItemObject, room RoomObject) {
}
```

`onLost()` is called when a player loses something from their backpack, such as dropping it.

|  Argument | Explanation |
| --- | --- |
| user | [ActorObject](FUNCTIONS_ACTORS.md) |
| item | [ItemObject](FUNCTIONS_ITEMS.md) |
| room | [RoomObject](FUNCTIONS_ROOMS.md) |


_Note: changes to the item will not be saved from this event._

---

```
function onFound(user ActorObject, item ItemObject, room RoomObject) {
}
```

`onFound()` is called when a player gains an item in their backpack.

|  Argument | Explanation |
| --- | --- |
| user | [ActorObject](FUNCTIONS_ACTORS.md) |
| item | [ItemObject](FUNCTIONS_ITEMS.md) |
| room | [RoomObject](FUNCTIONS_ROOMS.md) |

---

```
function onCommand(cmd string, user ActorObject, item ItemObject, room RoomObject) {
}
```

`onCommand()` is called any time a player types any command targetting a matching item name (even invalid commands).

For example, `feel bag`.

Returning `true` will halt any further processing of the response (i.e. "I've handled it"), and returning `false` will all the command to continue along and be processed as normal.

|  Argument | Explanation |
| --- | --- |
| cmd | the command entered, such as `rub`, `touch` or `activate`. |
| user | [ActorObject](FUNCTIONS_ACTORS.md) |
| item | [ItemObject](FUNCTIONS_ITEMS.md) |
| room | [RoomObject](FUNCTIONS_ROOMS.md) |

---

```
function onCommand_{command}(user ActorObject, item ItemObject, room RoomObject) {
}
```

`onCommand_{command}()` is called if a player types whatever is after the underscore, followed by a matching item name.

For example, `onCommand_feel()` would be called if anyone types `feel bag`.

If an `onCommand_{command}` is defined in a script, that command will not be passed to the normal `onCommand()`. So `onCommand_feel()` would be called, but `onCommand()` with `feel` as a `cmd` value would not.

In all other ways, this follows the same rules as the normal `onCommand()` function.

|  Argument | Explanation |
| --- | --- |
| user | [ActorObject](FUNCTIONS_ACTORS.md) |
| item | [ItemObject](FUNCTIONS_ITEMS.md) |
| room | [RoomObject](FUNCTIONS_ROOMS.md) |

---

```
function onPurchase(user ActorObject, item ItemObject, room RoomObject) {
}
```

`onPurchase()` is called when a player successfully purchases an item in a shop, after the item is

|  Argument | Explanation |
| --- | --- |
| user | [ActorObject](FUNCTIONS_ACTORS.md) |
| item | [ItemObject](FUNCTIONS_ITEMS.md) |
| room | [RoomObject](FUNCTIONS_ROOMS.md) |

Returning `false` will prevent the user from being given the object. This can be useful if you want to initate a script from a purchase, but not actually grant the item to the user, such as purchasing a ticket for travel, renting a room, or accessing a gated quest area.

---