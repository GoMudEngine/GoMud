// The Alchemist's Workshop - Room 111
// Alchemist Yenna explains potion-making and advances the alchemy quest step.

const alchemistMobId = 53;

function onEnter(user, room) {

    // Only trigger if player has alchemy quest and hasn't completed it
    if ( !user.HasQuest("1-alchemy") || user.HasQuest("1-wilderness") ) {
        return false;
    }

    var alchemist = room.GetMob(alchemistMobId, true);
    if ( alchemist == null ) {
        return false;
    }

    alchemist.Command('say Alchemy is pattern recognition. The world has rules and compounds have relationships.', 1.0);
    alchemist.Command('say A healing poultice is crushed desert sage, rendered fat, and a pinch of mineral salt from the basin rim. The sage draws contamination, the fat carries it, the salt seals the wound.', 2.5);
    alchemist.Command('say Everything in that formulation has a reason. If you swap components without understanding why, you get something useless at best, poisonous at worst.', 4.5);
    alchemist.Command('say The world is full of things that want to kill you. The ones that can also heal you are the most interesting.', 6.0);
    alchemist.Command('say That is the beginning of alchemy. Go find Fen in the meadow -- they will teach you where the raw materials come from.', 7.5);

    // Advance wilderness quest
    user.GiveQuest("1-wilderness");

    return false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var alchemist = room.GetMob(alchemistMobId, true);
        if ( alchemist != null ) {
            if ( user.HasQuest("1-alchemy") && !user.HasQuest("1-wilderness") ) {
                alchemist.Command('say Pay attention. I do not repeat myself more than twice.');
            } else if ( user.HasQuest("1-wilderness") ) {
                alchemist.Command('say You have your introduction. Fen is in the West Meadow.');
            } else {
                alchemist.Command('say Come back when you have spoken with the Blacksmith.');
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
