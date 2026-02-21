// West Meadow - Room 106
// Wilderness Guide Fen explains foraging/tracking and advances the wilderness quest step.

const guideMobId = 54;

function onEnter(user, room) {

    // Only trigger if player has done alchemy but not yet wilderness
    if ( !user.HasQuest("1-alchemy") || user.HasQuest("1-wilderness") ) {
        return true;
    }

    var guide = room.GetMob(guideMobId, true);
    if ( guide == null ) {
        return true;
    }

    guide.Command('emote straightens up from examining a patch of scrub.', 1.0);
    guide.Command('say Survival in the open country is about reading what is already there.', 2.0);
    guide.Command('say You do not fight the desert. You learn its logic. Water collects where the basalt dips. Shelter is where the largest outcrops cluster. Food is wherever small creatures are active.', 3.5);
    guide.Command('say The grey-green brush here -- desert sage. Edible, medicinal if prepared correctly, and it tells you that this soil retains some moisture beneath the surface.', 5.5);
    guide.Command('say Try it now. Type <ansi fg="command">forage</ansi> and the land will give you what it has. Not always much -- but something.', 7.0);
    guide.Command('say Tracking is the same principle. Something passed here. The ground tells you when, how heavy, and how fast, if you know the language. Type <ansi fg="command">track</ansi> to read it.', 8.5);
    guide.Command('say One more thing. In the open, being seen is a choice. Type <ansi fg="command">sneak</ansi> before you move and you become harder to notice -- useful when you would rather observe than be observed.', 10.5);
    guide.Command('say You have the fundamentals. The world is text, and you are learning to read it. The Elder is above the Alchemist, at the Observatory.', 12.5);

    // Advance quest to wilderness
    user.GiveQuest("1-wilderness");

    return true;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var guide = room.GetMob(guideMobId, true);
        if ( guide != null ) {
            if ( user.HasQuest("1-alchemy") && !user.HasQuest("1-wilderness") ) {
                guide.Command('say Watch the ground first. It tells you everything.');
            } else if ( user.HasQuest("1-wilderness") ) {
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
