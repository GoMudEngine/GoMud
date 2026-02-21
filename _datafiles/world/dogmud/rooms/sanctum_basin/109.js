// The Forge - Room 109
// Blacksmith Korvath explains forging and advances the crafting quest step.

const smithMobId = 52;

function onEnter(user, room) {

    // Only trigger if player has crafting quest and hasn't completed it
    if ( !user.HasQuest("1-crafting") || user.HasQuest("1-alchemy") ) {
        return false;
    }

    var smith = room.GetMob(smithMobId, true);
    if ( smith == null ) {
        return false;
    }

    smith.Command('emote sets down the hammer and turns to face you.', 1.0);
    smith.Command('say Crafting is belief made material. You take raw substance and you impose a shape on it through skill and intent.', 2.0);
    smith.Command('say The forge does not care about your feelings. It cares about your technique.', 3.5);
    smith.Command('say Heat the metal until it tells you it is ready -- color, texture, the sound it makes. Then shape it before doubt sets in.', 5.0);
    smith.Command('say You will learn the rest through practice. The fundamentals are in you now. Move on.', 6.5);

    // Advance crafting quest
    user.GiveQuest("1-alchemy");

    return false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var smith = room.GetMob(smithMobId, true);
        if ( smith != null ) {
            if ( user.HasQuest("1-crafting") && !user.HasQuest("1-alchemy") ) {
                smith.Command('say Listen carefully. Craft is not taught with words.');
            } else if ( user.HasQuest("1-alchemy") ) {
                smith.Command('say You have what you need from me. The Alchemist is west of the well.');
            } else {
                smith.Command('say Complete your combat training first. Come back when the trainer has cleared you.');
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
