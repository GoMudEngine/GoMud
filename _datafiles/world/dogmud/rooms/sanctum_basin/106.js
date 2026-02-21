// West Meadow - Room 106
// Wilderness Guide Fen explains foraging/tracking and advances the wilderness quest step.

const guideMobId = 54;

function onEnter(user, room) {

    // Only trigger if player has wilderness quest and hasn't completed it
    if ( !user.HasQuest("1-wilderness") || user.HasQuest("1-magic") ) {
        return false;
    }

    var guide = room.GetMob(guideMobId, true);
    if ( guide == null ) {
        return false;
    }

    guide.Command('emote straightens up from examining a patch of scrub.', 1.0);
    guide.Command('say Survival in the open country is about reading what is already there.', 2.0);
    guide.Command('say You do not fight the desert. You learn its logic. Water collects where the basalt dips. Shelter is where the largest outcrops cluster. Food is wherever small creatures are active.', 3.5);
    guide.Command('say The grey-green brush here -- desert sage. Edible, medicinal if prepared correctly, and it tells you that this soil retains some moisture beneath the surface.', 5.5);
    guide.Command('say Tracking is the same principle. Something passed here. The ground tells you when, how heavy, and how fast, if you know the language.', 7.0);
    guide.Command('say You have the fundamentals. The world is text, and you are learning to read it. The Elder is above the Alchemist, at the Observatory.', 8.5);

    // Advance magic quest
    user.GiveQuest("1-magic");

    return false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var guide = room.GetMob(guideMobId, true);
        if ( guide != null ) {
            if ( user.HasQuest("1-wilderness") && !user.HasQuest("1-magic") ) {
                guide.Command('say Watch the ground first. It tells you everything.');
            } else if ( user.HasQuest("1-magic") ) {
                guide.Command('say The Elder is waiting at the Observatory. North through the Alchemist, then up.');
            } else {
                guide.Command('say Come back when the Alchemist has spoken with you.');
            }
        }
        return false;
    }
    return false;
}

function onExit(user, room) {
}

function onLoad(room) {
}
