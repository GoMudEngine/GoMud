// Basin Gate - Room 102
// Basin Warden checks quest state and responds accordingly.

const wardenMobId = 56;

function onEnter(user, room) {

    var warden = room.GetMob(wardenMobId, true);
    if ( warden == null ) {
        return true;
    }

    if ( user.HasQuest("1-end") ) {
        // Already graduated -- brief acknowledgment, open gate quietly
        warden.Command('emote gives a brief nod of recognition.', 0.5);
        warden.Command('say Gate is open to you. Safe travels out there.', 1.5);
        room.SetLocked("south", false);

    } else if ( user.HasQuest("1-cave") ) {
        // First-time graduation -- full ceremony, give rewards
        warden.Command('say Your record is complete. Six trials, six instructors -- and the cave.', 1.0);
        warden.Command('say The Basin Warden has one function: to ensure that no one leaves Sanctum Basin unprepared. You are prepared.', 2.5);
        warden.Command('say The gate is open. Whatever you find south of here -- remember what you learned in the basin.', 4.0);
        warden.Command('say One last thing: the world is larger than what six instructors can cover. Type <ansi fg="command">help</ansi> to see what documentation is available -- there is more to learn out there than what we teach here.', 5.5);
        warden.Command('say The south road reaches Confluence within a day\'s walk. Follow the river north from there and you will find New Plymouth -- the largest settlement in this part of Gaius. That is where most people head first.', 7.0);
        warden.Command('emote steps aside and unlatches the gate with a single practiced motion.', 9.0);
        room.SetLocked("south", false);
        user.GiveQuest("1-end");

    } else if ( !user.HasQuest("1-start") ) {
        // Hasn't started trials at all
        warden.Command('say Hold.', 1.0);
        warden.Command('say You have not begun the Sanctum Trials. Speak with the Chrysalis Priest in the Academy Hall -- north to Town Square, then north and east.', 2.0);
        room.SetLocked("south", true);

    } else {
        // In progress -- short check-in, no lecture
        warden.Command('emote glances at you and returns their gaze to the road.', 0.5);
        warden.Command('say Not yet. Finish your trials and come back.', 1.5);
        warden.Command('say Type <ansi fg="command">quest</ansi> if you have lost your way.', 2.5);
        room.SetLocked("south", true);
    }

    return true;
}

function onExit(user, room) {
    // Relock the gate after the player passes through -- each player earns their own exit
    room.SetLocked("south", true);
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var warden = room.GetMob(wardenMobId, true);
        if ( warden != null ) {
            if ( user.HasQuest("1-end") ) {
                warden.Command('say The world is wide. Come back if you need to.');
            } else if ( user.HasQuest("1-cave") ) {
                warden.Command('say The gate is open. Move when you are ready.');
            } else if ( user.HasQuest("1-start") ) {
                warden.Command('say Your trials are not complete. Type <ansi fg="command">quest</ansi> to see what remains.');
            } else {
                warden.Command('say Find the Chrysalis Priest in the Academy Hall. North to Town Square, then north and east.');
            }
        }
        return false;
    }
    return false;
}

function onLoad(room) {
    // Gate starts locked
    room.SetLocked("south", true);
}
