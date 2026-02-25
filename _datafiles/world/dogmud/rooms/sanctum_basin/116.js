// Observatory - Room 116
// Elder Saris explains spellcasting (bifurcation model), introduces spell discovery, sends player to cave.

const elderMobId = 55;

function onEnter(user, room) {

    // Only trigger if player has done wilderness but not yet magic
    if ( !user.HasQuest("1-wilderness") || user.HasQuest("1-magic") ) {
        return true;
    }

    var elder = room.GetMob(elderMobId, true);
    if ( elder == null ) {
        return true;
    }

    elder.Command('emote turns from the sighting device without haste.', 1.0);
    elder.Command('say The old records call them the Witnesses -- three moons, each on a different cycle. Swiftmoon turns in under five days. The Wanderer takes ten. The Eye takes twenty-one.', 2.0);
    elder.Command('say Spellcasting begins with precision. Form a clear image of what you want to be true. Not a wish. Not a hope. An image. Hold it precisely.', 4.5);
    elder.Command('say Then bifurcate it. Split your inner vision into two identical copies of that image and hold both simultaneously. Then split again. Four. Eight.', 7.0);
    elder.Command('say Each doubling is a fold. Simple effects require only a few folds. Complex ones require many more. You will learn to hold more with practice.', 9.5);
    elder.Command('say The more you cast, the more patterns crystallize in your mind. New spells emerge from practice -- not from books or teachers. The mind teaches itself.', 11.5);
    elder.Command('say You already carry the seed of one pattern: Conviction Spike. Use it. Cast often. More will come.', 13.5);
    elder.Command('say The last thing standing between you and the south gate is the cave system below. The lichen down there gives off enough light to see by, but the Aberrant at the back does not respond to reason.', 15.5);
    elder.Command('say If you have questions about this world, the Chrysalis, the moons, the Fold -- <ansi fg="command">ask</ansi> me. I have had forty years to think.', 17.5);

    // Advance quest to magic step (no longer teaches illum -- spells are discovered through casting)
    user.GiveQuest("1-magic");

    return true;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var elder = room.GetMob(elderMobId, true);
        if ( elder != null ) {
            if ( user.HasQuest("1-wilderness") && !user.HasQuest("1-magic") ) {
                elder.Command('say The moons have been patient. I can afford the same courtesy. If you have questions, <ansi fg="command">ask</ansi> them.');
            } else if ( user.HasQuest("1-cave") ) {
                elder.Command('say The Warden waits at the south gate. You have earned the road. If anything is still unclear, <ansi fg="command">ask</ansi>.');
            } else if ( user.HasQuest("1-magic") ) {
                elder.Command('say The cave is below. The lichen will light your way. The Aberrant waits at the back -- it does not negotiate. <ansi fg="command">Ask</ansi> if you need guidance.');
            } else {
                elder.Command('say Complete the other trials first. The work matters. <ansi fg="command">Ask</ansi> if you have questions about this place.');
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
