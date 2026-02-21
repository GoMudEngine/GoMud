// Academy Hall - Room 113
// Chrysalis Greeter delivers intro lore and starts the Sanctum Trials quest.

const greeterMobId = 50;

function onEnter(user, room) {

    // Only trigger intro if player hasn't received it yet
    if ( user.HasQuest("1-arrive") ) {
        return false;
    }

    var greeter = room.GetMob(greeterMobId, true);
    if ( greeter == null ) {
        return false;
    }

    greeter.Command('say Welcome to Sanctum Basin.', 1.0);
    greeter.Command('say This plateau is a waystation -- a place where those newly arrived in this world learn what they need to survive it.', 2.0);
    greeter.Command('say The world of Gaius runs on belief. Belief shapes the land, animates the creatures that inhabit it, and determines the character of those who walk it.', 3.5);
    greeter.Command('say The Chrysalis is the form between what was and what will be. That is what you are, right now.', 5.0);
    greeter.Command('say Before the Basin Warden will open the south gate, you must complete six trials -- one with each of our instructors.', 6.5);
    greeter.Command('say Combat training is east of here, in the Training Yard. The Forge is further east. The Alchemist is west of the well.', 8.0);
    greeter.Command('say The Wilderness Guide works the West Meadow. The Elder watches from the Observatory above the Alchemist.', 9.5);
    greeter.Command('say When you have completed all six, return to the Basin Warden at the south gate. Safe travels.', 11.0);

    // Mark arrival step complete and give combat step
    user.GiveQuest("1-arrive");
    user.GiveQuest("1-combat");

    return false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" || cmd == "greet" ) {
        var greeter = room.GetMob(greeterMobId, true);
        if ( greeter != null ) {
            if ( user.HasQuest("1-arrive") ) {
                greeter.Command('say You have already received the briefing. Visit each instructor and complete your trials.');
            } else {
                greeter.Command('say Welcome to Sanctum Basin. Speak with me and I will explain what awaits you here.');
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
