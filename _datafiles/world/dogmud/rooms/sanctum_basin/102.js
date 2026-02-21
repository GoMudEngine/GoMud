// Basin Gate - Room 102
// Basin Warden checks for 1-graduate quest and unlocks/relocks south exit accordingly.

const wardenMobId = 56;

function onEnter(user, room) {

    var warden = room.GetMob(wardenMobId, true);
    if ( warden == null ) {
        return false;
    }

    if ( user.HasQuest("1-graduate") ) {
        // Player has completed all trials - unlock and congratulate
        warden.Command('say Your record is complete. Six trials, six instructors -- all confirmed.', 1.0);
        warden.Command('say The Basin Warden has one function: to ensure that no one leaves Sanctum Basin unprepared. You are prepared.', 2.5);
        warden.Command('say The gate is open. Whatever you find south of here -- remember what you learned in the basin.', 4.0);
        warden.Command('emote steps aside and unlatches the gate with a single practiced motion.', 5.0);

        room.SetLocked("south", false);

    } else {
        // Player has not completed trials - warn and keep locked
        warden.Command('say Hold.', 1.0);

        if ( !user.HasQuest("1-arrive") ) {
            warden.Command('say You have not begun the Sanctum Trials. Speak with the Chrysalis Greeter in the Academy Hall to the north.', 2.0);
        } else {
            warden.Command('say The gate remains closed until you have completed all six trials.', 2.0);
            warden.Command('say Visit the Training Yard, the Forge, the Alchemist, the West Meadow, and the Observatory if you have not already done so.', 3.5);
            warden.Command('say Return when your record is complete.', 5.0);
        }

        room.SetLocked("south", true);
    }

    return false;
}

function onExit(user, room) {
    // Relock the gate after the player passes through -- each player earns their own exit
    room.SetLocked("south", true);
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var warden = room.GetMob(wardenMobId, true);
        if ( warden != null ) {
            if ( user.HasQuest("1-graduate") ) {
                warden.Command('say The gate is open. Move when you are ready.');
            } else {
                warden.Command('say Complete your trials. Return when all six instructors have cleared you.');
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
