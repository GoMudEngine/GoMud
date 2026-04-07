// Observatory - Room 116
// Elder Saris explains spellcasting, player casts on Chrysalis Echo.
// Sub-steps: magic_arrive -> magic_cast

const elderMobId = 55;
const echoMobId = 112;

var castAttempted = false;
var castTicks = 0;

function onEnter(user, room) {

    // Only trigger if player has done wilderness_track but not yet magic_arrive
    if ( !user.HasQuest("1-wilderness_track") || user.HasQuest("1-magic_arrive") ) {
        return true;
    }

    var elder = room.GetMob(elderMobId, true);
    if ( elder == null ) {
        return true;
    }

    // Ensure the Echo is spawned
    var echo = room.GetMob(echoMobId, true);

    elder.Command('emote turns from the sighting device without haste.', 1.0);
    elder.Command('say The old records call them the Witnesses -- three moons, each on a different cycle. Swiftmoon turns in under five days. The Wanderer takes ten. The Eye takes twenty-one.', 2.0);
    elder.Command('say Spellcasting begins with precision. Form a clear image of what you want to be true. Not a wish. Not a hope. An image. Hold it precisely.', 4.5);
    elder.Command('say Then bifurcate it. Split your inner vision into two identical copies and hold both simultaneously. Then split again. Four. Eight.', 7.0);
    elder.Command('say Each doubling is a fold. Simple effects require only a few folds. Complex ones require many more.', 9.0);
    elder.Command('say The more you cast, the more patterns crystallize in your mind. New spells emerge from practice -- not from books or teachers.', 11.0);
    elder.Command('say You already carry the seed of one pattern: Conviction Spike.', 13.0);
    elder.Command('emote gestures toward the faintly shimmering figure nearby.', 14.5);
    elder.Command('say That is a Chrysalis Echo -- a harmless remnant of a past Awakening. I keep it here for practice.', 15.5);
    elder.Command('say Try your spell on it. Type <ansi fg="command">cast conviction-spike echo</ansi>.', 17.0);

    user.GiveQuest("1-magic_arrive");

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Detect cast command — works whether player types "cast conviction-spike echo",
    // "conviction-spike echo" (spell ID shortcut), or uses an alias.
    if ( (cmd == "cast" && rest != "") || cmd == "conviction-spike" ) {
        if ( user.HasQuest("1-magic_arrive") && !user.HasQuest("1-magic_cast") ) {
            castAttempted = true;
            castTicks = 0;
        }
    }

    if ( cmd == "talk" ) {
        var elder = room.GetMob(elderMobId, true);
        if ( elder != null ) {
            if ( user.HasQuest("1-magic_arrive") && !user.HasQuest("1-magic_cast") ) {
                elder.Command('say The Echo is waiting. Type <ansi fg="command">cast conviction-spike echo</ansi>.');
            } else if ( user.HasQuest("1-cave") ) {
                elder.Command('say The Warden waits at the south gate. You have earned the road. If anything is still unclear, <ansi fg="command">ask</ansi>.');
            } else if ( user.HasQuest("1-magic_cast") ) {
                elder.Command('say The cave is below. The lichen will light your way. The Aberrant waits at the back -- it does not negotiate. <ansi fg="command">Ask</ansi> if you need guidance.');
            } else {
                elder.Command('say Complete the other trials first. The work matters. <ansi fg="command">Ask</ansi> if you have questions about this place.');
            }
        }
        return false;
    }
    return false;
}

function onIdle(room) {

    if ( !castAttempted ) {
        return;
    }

    castTicks++;
    if ( castTicks < 2 ) {
        return;
    }

    castAttempted = false;
    castTicks = 0;

    var players = room.GetPlayers();
    if ( players == null ) {
        return;
    }

    for ( var i = 0; i < players.length; i++ ) {
        var player = players[i];
        if ( player.HasQuest("1-magic_arrive") && !player.HasQuest("1-magic_cast") ) {
            var elder = room.GetMob(elderMobId, true);
            if ( elder != null ) {
                elder.Command('emote watches the air settle after the discharge.', 0.5);
                elder.Command('say There. You felt it -- the pattern forming, the bifurcation holding.', 2.0);
                elder.Command('say Cast often. More patterns will crystallize with practice.', 3.5);
                elder.Command('say The last thing standing between you and the south gate is the cave system below. The Aberrant at the back does not respond to reason.', 5.0);
                elder.Command('say If you have questions about this world, the Chrysalis, the moons, the Fold -- <ansi fg="command">ask</ansi> me.', 6.5);
            }
            player.GiveQuest("1-magic_cast");
        }
    }
}

function onExit(user, room) {
}

function onLoad(room) {
}
