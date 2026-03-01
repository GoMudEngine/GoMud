// Basin Gate - Room 102
// Basin Warden checks quest state and gives appropriate dialogue.
// Steps: cave -> warden -> end
// No lock -- the gate is always passable, but the warden reacts to
// the player's quest state and grants completion quests when earned.

const wardenMobId = 56;

function onEnter(user, room) {

    var warden = room.GetMob(wardenMobId, true);
    if ( warden == null ) {
        return true;
    }

    if ( user.HasQuest("1-end") ) {
        // Already graduated -- brief acknowledgment
        warden.Command('emote gives a brief nod of recognition.', 0.5);
        warden.Command('say Safe travels out there.', 1.5);

    } else if ( user.HasQuest("1-warden") ) {
        // Already got warden speech but hasn't left yet
        warden.Command('emote nods toward the open gate.', 0.5);

    } else if ( user.HasQuest("1-cave") ) {
        // First-time graduation -- warden speech, then end step
        warden.Command('say Your record is complete. Six trials, six instructors -- and the cave.', 1.0);
        warden.Command('say The Basin Warden has one function: to ensure that no one leaves Sanctum Basin unprepared. You are prepared.', 2.5);
        warden.Command('say The gate is open. Whatever you find south of here -- remember what you learned in the basin.', 4.0);
        warden.Command('say One last thing: the world is larger than what six instructors can cover. Type <ansi fg="command">help</ansi> to see what documentation is available -- there is more to learn out there than what we teach here.', 5.5);
        warden.Command('say The south road reaches Confluence within a day\'s walk. Follow the river north from there and you will find New Plymouth -- the largest settlement in this part of Gaius. That is where most people head first.', 7.0);
        warden.Command('emote steps aside and gestures toward the road south.', 9.0);
        user.GiveQuest("1-warden");
        user.GiveQuest("1-end");

    } else if ( !user.HasQuest("1-start") ) {
        // Hasn't started trials at all
        warden.Command('say Hold.', 1.0);
        warden.Command('say You have not begun the Sanctum Trials. Speak with the Chrysalis Priest in the Academy Hall -- north to Town Square, then north and east.', 2.0);

    } else {
        // In progress -- short check-in
        warden.Command('emote glances at you and returns their gaze to the road.', 0.5);
        warden.Command('say Not yet. Finish your trials and come back.', 1.5);
        warden.Command('say Type <ansi fg="command">quest</ansi> if you have lost your way.', 2.5);
    }

    return true;
}

function onExit(user, room) {
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
}
