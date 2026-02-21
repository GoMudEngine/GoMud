// Observatory - Room 116
// Elder Saris explains the Fold (bifurcation model), teaches illuminate, sends player to cave.

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
    elder.Command('say The old records call them the Witnesses -- three moons, each on a different cycle. Swiftmoon turns in under five days. The Wanderer takes ten. The Eye takes twenty-one. Their combined position determines what we call Fold pressure.', 2.0);
    elder.Command('say On Gaius, belief and physics are less separate than people prefer to think. What we call the Fold is where they meet -- the substrate that intent acts upon.', 4.5);
    elder.Command('say Spellcasting is the technique for acting on it deliberately. The foundation is this: form a clear image of what you want to be true. Not a wish. Not a hope. An image. Light appearing here, around you. Hold it precisely.', 6.5);
    elder.Command('say Then bifurcate it. Split your inner vision into two identical copies of that image and hold both simultaneously. Then split again. Four. Eight.', 9.0);
    elder.Command('say Each doubling is a fold. The Fold responds to the accumulated weight of that sustained conviction. Illuminate requires holding three folds without collapse. You can learn to hold three.', 11.0);
    elder.Command('say More complex effects require more folds. Most people lose coherence at five or six. The Witnesses affect the ceiling -- higher pressure nights allow more folds before the mind gives way.', 13.0);
    elder.Command('say That pattern is in your mind now. Type <ansi fg="command">spells</ansi> to see what you know. You will find illuminate there.', 15.0);
    elder.Command('say The last thing standing between you and the south gate is the cave system below. Take the light with you -- it is dark in there. The Aberrant at the back does not respond to reason, but it will respond to you.', 17.0);

    // Teach illuminate spell and advance quest to magic step
    user.LearnSpell("illum");
    user.GiveQuest("1-magic");

    return true;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var elder = room.GetMob(elderMobId, true);
        if ( elder != null ) {
            if ( user.HasQuest("1-wilderness") && !user.HasQuest("1-magic") ) {
                elder.Command('say The moons have been patient. I can afford the same courtesy.');
            } else if ( user.HasQuest("1-cave") ) {
                elder.Command('say The Warden waits at the south gate. You have earned the road.');
            } else if ( user.HasQuest("1-magic") ) {
                elder.Command('say The cave is below. Light your way with illuminate and find what is waiting at the back.');
            } else {
                elder.Command('say Complete the other trials first. The Fold rewards preparation.');
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
