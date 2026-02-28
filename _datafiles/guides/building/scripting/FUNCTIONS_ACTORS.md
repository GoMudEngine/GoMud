# ActorObject

ActorObjects are the basic object that represents Users and NPCs

- [ActorObject](#actorobject)
  - [ActorNames(actors \[\]ActorObject) string ](#actornamesactors-actorobject-string-)
  - [GetUser(userId int) ActorObject ](#getuseruserid-int-actorobject-)
  - [GetMob(mobInstanceId int \[,createIfMissing bool\]) ActorObject ](#getmobmobinstanceid-int-createifmissing-bool-actorobject-)
  - [ActorObject.UserId() int](#actorobjectuserid-int)
  - [ActorObject.InstanceId() int](#actorobjectinstanceid-int)
  - [ActorObject.MobTypeId() int](#actorobjectmobtypeid-int)
  - [ActorObject.SendText(msg string)](#actorobjectsendtextmsg-string)
  - [RoomObject.SendText(msg string)](#roomobjectsendtextmsg-string)
  - [ActorObject.GetSize() string](#actorobjectgetsize-string)
  - [ActorObject.GetStat(statName string) int](#actorobjectgetstatstatname-string-int)
  - [ActorObject.SetTempData(key string, value any)](#actorobjectsettempdatakey-string-value-any)
  - [ActorObject.GetTempData(key string) any](#actorobjectgettempdatakey-string-any)
  - [ActorObject.SetMiscCharacterData(key string, value any)](#actorobjectsetmisccharacterdatakey-string-value-any)
  - [ActorObject.GetMiscCharacterData(key string) any](#actorobjectgetmisccharacterdatakey-string-any)
  - [ActorObject.GetMiscCharacterDataKeys(\[ prefix1, prefix2 \]) \[\]string](#actorobjectgetmisccharacterdatakeys-prefix1-prefix2--string)
  - [ActorObject.GetCharacterName( wrapInTags bool ) string](#actorobjectgetcharactername-wrapintags-bool--string)
  - [ActorObject.SetCharacterName( newName string )](#actorobjectsetcharactername-newname-string-)
  - [ActorObject.GetRoomId() int](#actorobjectgetroomid-int)
  - [ActorObject.HasQuest(questId string) bool](#actorobjecthasquestquestid-string-bool)
  - [ActorObject.GiveQuest(questId string)](#actorobjectgivequestquestid-string)
  - [ActorObject.GetPartyMembers() \[\]Actor](#actorobjectgetpartymembers-actor)
  - [ActorObject.AddGold(amt int \[, bankAmt int\])](#actorobjectaddgoldamt-int--bankamt-int)
  - [ActorObject.AddHealth(amt int) int](#actorobjectaddhealthamt-int-int)
  - [ActorObject.Sleep(seconds int)](#actorobjectsleepseconds-int)
  - [ActorObject.Command(cmd string \[, waitTurns int\])](#actorobjectcommandcmd-string--waitturns-int)
  - [ActorObject.CommandFlagged(cmd string, flag int \[, waitTurns int\])](#actorobjectcommandflaggedcmd-string-flag-int--waitturns-int)
  - [ActorObject.IsTameable() bool](#actorobjectistameable-bool)
  - [ActorObject.TrainSkill(skillName string, skillLevel int)](#actorobjecttrainskillskillname-string-skilllevel-int)
  - [ActorObject.GetSkillLevel(skillName string)](#actorobjectgetskilllevelskillname-string)
  - [ActorObject.MoveRoom(destRoomId int \[, leaveCharmedMobsBehind bool\] )](#actorobjectmoveroomdestroomid-int--leavecharmedmobsbehind-bool-)
  - [ActorObject.UpdateItem(itemId ItemObject)](#actorobjectupdateitemitemid-itemobject)
  - [ActorObject.GiveItem(itemId ItemObject)](#actorobjectgiveitemitemid-itemobject)
  - [ActorObject.TakeItem(itemId ItemObject)](#actorobjecttakeitemitemid-itemobject)
  - [ActorObject.HasBuff(buffId int) bool](#actorobjecthasbuffbuffid-int-bool)
  - [ActorObject.GiveBuff(buffId int, source string)](#actorobjectgivebuffbuffid-int-source-string)
  - [ActorObject.HasBuffFlag(buffFlag string) bool](#actorobjecthasbuffflagbuffflag-string-bool)
  - [ActorObject.CancelBuffWithFlag(buffFlag string) bool](#actorobjectcancelbuffwithflagbuffflag-string-bool)
  - [ActorObject.RemoveBuff(buffId int)](#actorobjectremovebuffbuffid-int)
  - [ActorObject.HasItemId(itemId int, \[excludeWorn bool\]) bool](#actorobjecthasitemiditemid-int-excludeworn-bool-bool)
  - [ActorObject.GetBackpackItems() \[\]ItemObject](#actorobjectgetbackpackitems-itemobject)
  - [ActorObject.HasSpell(spellId string)](#actorobjecthasspellspellid-string)
  - [ActorObject.LearnSpell(spellId string) bool](#actorobjectlearnspellspellid-string-bool)
  - [ActorObject.IsAggro(targetActor ActorObject)](#actorobjectisaggrotargetactor-actorobject)
  - [ActorObject.GetMobKills(mobId int) int](#actorobjectgetmobkillsmobid-int-int)
  - [ActorObject.GetRaceKills(raceName string) int](#actorobjectgetracekillsracename-string-int)
  - [ActorObject.GetHealth() int](#actorobjectgethealth-int)
  - [ActorObject.SetHealth(amt int)](#actorobjectsethealthamt-int)
  - [ActorObject.GetHealthMax() int](#actorobjectgethealthmax-int)
  - [ActorObject.GetHealthPct() float](#actorobjectgethealthpct-float)
  - [ActorObject.GetMana() int](#actorobjectgetmana-int)
  - [ActorObject.GetManaMax() int](#actorobjectgetmanamax-int)
  - [ActorObject.GetManaPct() float](#actorobjectgetmanapct-float)
  - [ActorObject.SetAdjective(adj string, addIt bool)](#actorobjectsetadjectiveadj-string-addit-bool)
  - [ActorObject.IsCharmed( \[userId1, userId2, etc \] ) bool ](#actorobjectischarmed-userid1-userid2-etc---bool-)
  - [ActorObject.GetCharmedUserId() int ](#actorobjectgetcharmeduserid-int-)
  - [ActorObject.CharmSet(userId int, charmRounds int, \[ onRevertCommand1, onRevertCommand2, etc \])](#actorobjectcharmsetuserid-int-charmrounds-int--onrevertcommand1-onrevertcommand2-etc-)
  - [ActorObject.CharmRemove()](#actorobjectcharmremove)
  - [ActorObject.CharmExpire()](#actorobjectcharmexpire)
  - [ActorObject.GetCharmCount() int](#actorobjectgetcharmcount-int)
  - [ActorObject.GetMaxCharmCount() int](#actorobjectgetmaxcharmcount-int)
  - [ActorObject.GiveExtraLife()](#actorobjectgiveextralife)
  - [ActorObject.ShorthandId() string](#actorobjectshorthandid-string)
  - [ActorObject.Uncurse()](#actorobjectuncurse)
  - [ActorObject.GetPet()](#actorobjectgetpet)
  - [ActorObject.GetLastInputRound() int](#actorobjectgetlastinputround-int)
  - [ActorObject.GetTameMastery() Object](#actorobjectgettamemastery-object)
  - [ActorObject.SetTameMastery(mobId int, newSkillLevel int)](#actorobjectsettamemasterymobid-int-newskilllevel-int)
  - [ActorObject.GetChanceToTame(target ScriptActor) int](#actorobjectgetchancetotametarget-scriptactor-int)
  - [ActorObject.GetStatMod(statModName string) int](#actorobjectgetstatmodstatmodname-string-int)
  - [ActorObject.IsHome() bool](#actorobjectishome-bool)
  - [ActorObject.Pathing() bool](#actorobjectpathing-bool)
  - [ActorObject.PathingAtWaypoint() bool](#actorobjectpathingatwaypoint-bool)
  - [ActorObject.TimerSet(name string, period string)](#actorobjecttimersetname-string-period-string)
  - [ActorObject.TimerExpired(name string) bool](#actorobjecttimerexpiredname-string-bool)
  - [ActorObject.TimerExists(name string) bool](#actorobjecttimerexistsname-string-bool)
  - [ActorObject.AddEventLog(category string, message string)](#actorobjectaddeventlogcategory-string-message-string)




## [ActorNames(actors []ActorObject) string ](/internal/scripting/actor_func.go)
Returns a formatted list of actor names, separated by commas, then "and".

_Example: "Tim, Jim and Henry"_

|  Argument | Explanation |
| --- | --- |
| actors | An array of ActorObjects. |

## [GetUser(userId int) ActorObject ](/internal/scripting/actor_func.go)
Retrieves a ActorObject for a given userId.

|  Argument | Explanation |
| --- | --- |
| userId | The target user id to get. |

## [GetMob(mobInstanceId int [,createIfMissing bool]) ActorObject ](/internal/scripting/actor_func.go)
Retrieves a ActorObject for a given mobInstanceId.

|  Argument | Explanation |
| --- | --- |
| mobInstanceId | The target mobInstanceId to get. |
| createIfMissing | If true and mob isn't found, the mob will be created and returned. |

## [ActorObject.UserId() int](/internal/scripting/actor_func.go)
Returns the userId of the ActorObject.˚

_Note: Only useful for User ActorObjects - Returns zero otherwise._

## [ActorObject.InstanceId() int](/internal/scripting/actor_func.go)
Returns the mobInstanceId of the ActorObject.

_Note: Only useful for Mob ActorObjects - Returns zero otherwise._

## [ActorObject.MobTypeId() int](/internal/scripting/actor_func.go)
Returns the base mobId used to spawn new instances.

_Note: Only useful for Mob ActorObjects - Returns zero otherwise._


## [ActorObject.SendText(msg string)](/internal/scripting/actor_func.go)
Sends a message to the actor.

|  Argument | Explanation |
| --- | --- |
| msg | the message to send |

## [RoomObject.SendText(msg string)](/internal/scripting/room_func.go)
Sends a message to everyone in the room.

|  Argument | Explanation |
| --- | --- |
| msg | the message to send |

## [ActorObject.GetSize() string](/internal/scripting/actor_func.go)
Returns `small`, `medium`, or `large`

## [ActorObject.GetStat(statName string) int](/internal/scripting/actor_func.go)
Returns the named stat value.

|  Argument | Explanation |
| --- | --- |
| statName | A stat name such as `strength`, `smarts`, `perception`, etc. |

## [ActorObject.SetTempData(key string, value any)](/internal/scripting/actor_func.go)
Sets temporary data for the ActorObject (Lasts until the ActorObject is removed from memory).

_Note: This is useful for saving/retrieving data that an ActorObject can carry along to multiple room scripts._

|  Argument | Explanation |
| --- | --- |
| key | A unique identifier for the data. |
| value | What you will be saving. If null, frees from memory. |

## [ActorObject.GetTempData(key string) any](/internal/scripting/actor_func.go)
Gets temporary data for the ActorObject.

_Note: This is useful for saving/retrieving data that a ActorObject can carry along to multiple room scripts._

|  Argument | Explanation |
| --- | --- |
| key | A unique identifier for the data. |

## [ActorObject.SetMiscCharacterData(key string, value any)](/internal/scripting/actor_func.go)
Sets permanent data for the ActorObject. 

_Note: This miscellaneous data is attached to the character data, not the user data. If the user changes characters, it will not follow._

_Note: There is a special key: `StartRoom` that will override the Start RoomId for the character if set._

|  Argument | Explanation |
| --- | --- |
| key | A unique identifier for the data. |
| value | What you will be saving. If null, frees from memory. |

## [ActorObject.GetMiscCharacterData(key string) any](/internal/scripting/actor_func.go)
Gets permanent data for the ActorObject.

_Note: This miscellaneous data is attached to the character data, not the user data. If the user changes characters, it will not follow._

|  Argument | Explanation |
| --- | --- |
| key | A unique identifier for the data. |

## [ActorObject.GetMiscCharacterDataKeys([ prefix1, prefix2 ]) []string](/internal/scripting/actor_func.go)
Gets a list of misc data keys for the ActorObject.

|  Argument | Explanation |
| --- | --- |
| prefix1, prefix2, etc | Optional strings of prefixes to return matching keys. |

## [ActorObject.GetCharacterName( wrapInTags bool ) string](/internal/scripting/actor_func.go)
Retrieves the name of a ActorObject.

|  Argument | Explanation |
| --- | --- |
| wrapInTags | If true, will return the name wrapped in ansi tags with the fg set to `username` or `mobname`. |

## [ActorObject.SetCharacterName( newName string )](/internal/scripting/actor_func.go)
Retrieves the name of a ActorObject.

|  Argument | Explanation |
| --- | --- |
| newName | The new name for the mob or player. |

## [ActorObject.GetRoomId() int](/internal/scripting/actor_func.go)
Returns the roomId a ActorObject is in.

## [ActorObject.HasQuest(questId string) bool](/internal/scripting/actor_func.go)
Get whether a ActorObject has a quest/progress.

|  Argument | Explanation |
| --- | --- |
| questId | The quest identifier string to check, such as `3-start`. |

## [ActorObject.GiveQuest(questId string)](/internal/scripting/actor_func.go)
Grants a quest or progress on a quest to a ActorObject. If they are in a party, grants to the party members as well.

|  Argument | Explanation |
| --- | --- |
| questId | The quest identifier string to give, such as `3-start`. |

## [ActorObject.GetPartyMembers() []Actor](/internal/scripting/actor_func.go)
Returns a list of actors in the party, both players and mobs.

## [ActorObject.AddGold(amt int [, bankAmt int])](/internal/scripting/actor_func.go)
Update how much gold an ActorObject has

|  Argument | Explanation |
| --- | --- |
| amt | A positive or negative amount of gold to alter the actors gold by. |
| bankAmt (optional) | A positive or negative amount of gold to alter the actors bank gold by. |

## [ActorObject.AddHealth(amt int) int](/internal/scripting/actor_func.go)
Update how much health an ActorObject has, and returns the actual amount their health changed.

|  Argument | Explanation |
| --- | --- |
| amt | A positive or negative amount of health to alter the actors health by. |


## [ActorObject.Sleep(seconds int)](/internal/scripting/actor_func.go)
Force a mob to wait this many seconds before executing any additional behaviors

_Note: Only works on mobs._

|  Argument | Explanation |
| --- | --- |
| seconds | How many seconds to wait. |

## [ActorObject.Command(cmd string [, waitTurns int])](/internal/scripting/actor_func.go)
Forces an ActorObject to execute a command as if they entered it

_Note: Don't underestimate the power of this function! Complex and interesting behaviors or interactions can emerge from using it._

|  Argument | Explanation |
| --- | --- |
| cmd | The command to execute such as `look west` or `say goodbye`. |
| waitTurns (optional) | The number of turns (NOT rounds) to wait before executing the command. |

## [ActorObject.CommandFlagged(cmd string, flag int [, waitTurns int])](/internal/scripting/actor_func.go)
Forces an ActorObject to execute a command as if they entered it.
WARNING: Advanced Usage. Required a flag integer.

_Note: Don't underestimate the power of this function! Complex and interesting behaviors or interactions can emerge from using it._

|  Argument | Explanation |
| --- | --- |
| cmd | The command to execute such as `look west` or `say goodbye`. |
| flag | The special control flag to pass to the command. |
| waitTurns (optional) | The number of turns (NOT rounds) to wait before executing the command. |

## [ActorObject.IsTameable() bool](/internal/scripting/actor_func.go)
Returns `true` if actor can be tamed.

## [ActorObject.TrainSkill(skillName string, skillLevel int)](/internal/scripting/actor_func.go)
Sets an ActorObject skill level, if it's greater than what they already have

|  Argument | Explanation |
| --- | --- |
| skillName | The name of the skill to train, such as `map` or `backstab`. |

## [ActorObject.GetSkillLevel(skillName string)](/internal/scripting/actor_func.go)
Returns the current skil level for the skillName, or zero if none.

|  Argument | Explanation |
| --- | --- |
| skillName | The name of the skill to train, such as `map` or `backstab`. |


## [ActorObject.MoveRoom(destRoomId int [, leaveCharmedMobsBehind bool] )](/internal/scripting/actor_func.go)
Quietly moves an ActorObject to a new room

|  Argument | Explanation |
| --- | --- |
| destRoomId | The room id to move them to. |
| leaveCharmedMobsBehind | If true, does not also move charmed mobs with the user. |

## [ActorObject.UpdateItem(itemId ItemObject)](/internal/scripting/actor_func.go)
Accepts an ItemObject to update in the players backpack. If the item does not already exist in the players backpack, it is ignored.

_Note: This is the only way to save changes made to an item in the players backpack._

|  Argument | Explanation |
| --- | --- |
| ItemObject | The item object to give them. |

## [ActorObject.GiveItem(itemId ItemObject)](/internal/scripting/actor_func.go)
Accepts an ItemObject to put into the players backpack. This can be called multiple times to duplicate an item.

|  Argument | Explanation |
| --- | --- |
| ItemObject | The item object to give them. |

## [ActorObject.TakeItem(itemId ItemObject)](/internal/scripting/actor_func.go)
Takes an object from the users backpack.

|  Argument | Explanation |
| --- | --- |
| ItemObject | The item object to take. |


## [ActorObject.HasBuff(buffId int) bool](/internal/scripting/actor_func.go)
Returns true if the Actor has the buffId supplied

|  Argument | Explanation |
| --- | --- |
| buffId | The ID of the buff to look for. |

## [ActorObject.GiveBuff(buffId int, source string)](/internal/scripting/actor_func.go)
Grants an ActorObject a Buff

|  Argument | Explanation |
| --- | --- |
| buffId | The ID of the buff to give them. |
| source | The source of the buff, "item", "spell", "trap", "curse", etc. or empty. |

## [ActorObject.HasBuffFlag(buffFlag string) bool](/internal/scripting/actor_func.go)
Find out if an ActorObject has a specific buff flag

|  Argument | Explanation |
| --- | --- |
| buffFlag | The buff flag to check [see buffspec.go](../buffs/buffspec.go). |

## [ActorObject.CancelBuffWithFlag(buffFlag string) bool](/internal/scripting/actor_func.go)
Cancels any buffs that have the flag provided. Returns `true` if one or more were found.

|  Argument | Explanation |
| --- | --- |
| buffFlag | The buff flag to check [see buffspec.go](../buffs/buffspec.go). |

## [ActorObject.RemoveBuff(buffId int)](/internal/scripting/actor_func.go)
Remove a buff without triggering onEnd

|  Argument | Explanation |
| --- | --- |
| buffId | The ID of the buff to remove |

## [ActorObject.HasItemId(itemId int, [excludeWorn bool]) bool](/internal/scripting/actor_func.go)
Check whether an ActorObject has an item id in their backpack

|  Argument | Explanation |
| --- | --- |
| itemId | The ItemId to check for. |
| itemId (optional) | Ignore worn items? |

## [ActorObject.GetBackpackItems() []ItemObject](/internal/scripting/actor_func.go)
Get a list of Item objects in the ActorObjects backpack

_Note: See [/scripting/docs/FUNCTIONS_ITEMS.md](FUNCTIONS_ITEMS.md) for details on ItemObject objects._

## [ActorObject.HasSpell(spellId string)](/internal/scripting/actor_func.go)
Returns true if the actor has the spell supplied

|  Argument | Explanation |
| --- | --- |
| spellId | The ID of the spell |

## [ActorObject.LearnSpell(spellId string) bool](/internal/scripting/actor_func.go)
Adds the spell to the Actors spellbook. Returns true if learned, false if already known.

|  Argument | Explanation |
| --- | --- |
| spellId | The ID of the spell |

## [ActorObject.IsAggro(targetActor ActorObject)](/internal/scripting/actor_func.go)
Returns true if the actor is aggro vs targetActor

|  Argument | Explanation |
| --- | --- |
| targetActor | [ActorObject](FUNCTIONS_ACTORS.md) |

## [ActorObject.GetMobKills(mobId int) int](/internal/scripting/actor_func.go)
Returns the number of times the actor has killed a certain mobId

|  Argument | Explanation |
| --- | --- |
| mobId | ID of the mob to check |

## [ActorObject.GetRaceKills(raceName string) int](/internal/scripting/actor_func.go)
Returns the number of times the actor has killed a certain race of mob

|  Argument | Explanation |
| --- | --- |
| raceName | race name such as human, goblin, rodent |

## [ActorObject.GetHealth() int](/internal/scripting/actor_func.go)
Returns current actor health

## [ActorObject.SetHealth(amt int)](/internal/scripting/actor_func.go)
Sets actor health to a specific amount. If this exceeds their maximum health, sets to their maximum health.

|  Argument | Explanation |
| --- | --- |
| amt | number of hitpoints to set them to |


## [ActorObject.GetHealthMax() int](/internal/scripting/actor_func.go)
Returns current actor max health

## [ActorObject.GetHealthPct() float](/internal/scripting/actor_func.go)
Returns current actor health as a percentage

## [ActorObject.GetMana() int](/internal/scripting/actor_func.go)
Returns current actor mana

## [ActorObject.GetManaMax() int](/internal/scripting/actor_func.go)
Returns current actor max mana

## [ActorObject.GetManaPct() float](/internal/scripting/actor_func.go)
Returns current actor mana as a percentage

## [ActorObject.SetAdjective(adj string, addIt bool)](/internal/scripting/actor_func.go)
Adds or removes a specific text adjective to the characters name

|  Argument | Explanation |
| --- | --- |
| adj | Adjective such as "sleeping", "crying" or "busy" |
| addIt | `true` to add it. `false` to remove it. |

## [ActorObject.IsCharmed( [userId1, userId2, etc ] ) bool ](/internal/scripting/actor_func.go)
Sets a mob to charmed by a user for a set number of rounds.

|  Argument | Explanation |
| --- | --- |
| userId | One or more users to test against. If ommitted, returns true if charmed at all by anyone. |

## [ActorObject.GetCharmedUserId() int ](/internal/scripting/actor_func.go)
Returns the userId that charmed this actor (or zero if none)

## [ActorObject.CharmSet(userId int, charmRounds int, [ onRevertCommand1, onRevertCommand2, etc ])](/internal/scripting/actor_func.go)
Sets a mob to charmed by a user for a set number of rounds.

|  Argument | Explanation |
| --- | --- |
| userId | userId that the mob will be charmed to |
| charmRounds | How many rounds it should last, or -1 for unlimited. |
| onRevertCommand | One or more commands for the mob to execute when the charm expires |

## [ActorObject.CharmRemove()](/internal/scripting/actor_func.go)
Immediately discards any charm effect without expiration effects.

## [ActorObject.CharmExpire()](/internal/scripting/actor_func.go)
Forces the current charm of the mob to expire

## [ActorObject.GetCharmCount() int](/internal/scripting/actor_func.go)
Returns the number of charmed creatures in the actors control

## [ActorObject.GetMaxCharmCount() int](/internal/scripting/actor_func.go)
Returns the maximum allowed charmed creatures for this actor

## [ActorObject.GiveExtraLife()](/internal/scripting/actor_func.go)
Increases extra lives by 1 for the player/actor

## [ActorObject.ShorthandId() string](/internal/scripting/actor_func.go)
Returns the shorthand ID string to refer to the mob or player ( `@123` or `#122` )

## [ActorObject.Uncurse()](/internal/scripting/actor_func.go)
Uncurses any objects the target has equipped

## [ActorObject.GetPet()](/internal/scripting/actor_func.go)
Returns the pet object for the actor, or null

## [ActorObject.GetLastInputRound() int](/internal/scripting/actor_func.go)
Returns the last round number the user input anything at all

## [ActorObject.GetTameMastery() Object](/internal/scripting/actor_func.go)
Returns an object where keys are the mobId and the value is the tame level

## [ActorObject.SetTameMastery(mobId int, newSkillLevel int)](/internal/scripting/actor_func.go)
Sets the tame mastery of a specific mobId to a specific skill level

|  Argument | Explanation |
| --- | --- |
| mobId | MobId for the type |
| newSkillLevel | New level to set it at |

## [ActorObject.GetChanceToTame(target ScriptActor) int](/internal/scripting/actor_func.go)
Get the chance in 100 to tame a target

|  Argument | Explanation |
| --- | --- |
| target | [ActorObject](FUNCTIONS_ACTORS.md) |

## [ActorObject.GetStatMod(statModName string) int](/internal/scripting/actor_func.go)
returns the total specific statmod from worn items and buffs

|  Argument | Explanation |
| --- | --- |
| statModName | The name of the special stat mod, such as "strength" or "tame" |

## [ActorObject.IsHome() bool](/internal/scripting/actor_func.go)
(mobs only) Returns true if the actor is at their home roomId

## [ActorObject.Pathing() bool](/internal/scripting/actor_func.go)
(mobs only) Returns true if actor is currently pathing

## [ActorObject.PathingAtWaypoint() bool](/internal/scripting/actor_func.go)
(mobs only) Returns true if actor is pathing and at a waypoint.


## [ActorObject.TimerSet(name string, period string)](/internal/scripting/actor_func.go)
Starts a new Round timer

|  Argument | Explanation |
| --- | --- |
| name | A string identifier. Reusing names will overwrite previously assigned names. |
| period | How long until the timer expires. `1 real hour`, `1 hour`, etc. |

## [ActorObject.TimerExpired(name string) bool](/internal/scripting/actor_func.go)
Returns true if the specified timer has expired or doesn't exist.

## [ActorObject.TimerExists(name string) bool](/internal/scripting/actor_func.go)
Returns true if the specified timer exists. 
Set timers always exist until they are checked for expiration with `TimerExpired(name string)`

## [ActorObject.AddEventLog(category string, message string)](/internal/scripting/actor_func.go)
Adds a line to the users Event Log (`history`)

|  Argument | Explanation |
| --- | --- |
| category | A short single word category  |
| message | A single line describing the event |