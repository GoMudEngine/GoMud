// The Forge - Room 109
// Blacksmith Korvath gives mats, player crafts iron dagger, Korvath reacts to outcome.
// Sub-steps: crafting_arrive -> crafting_craft

const smithMobId   = 52;
const ironIngotId  = 40001;
const leatherStripId = 40002;
const ironDaggerId = 10009;

var craftStarted    = false;
var craftIdleTicks  = 0;

function onEnter(user, room) {

    // Only trigger if player has done combat_defeat but not yet crafting_arrive
    if ( !user.HasQuest("1-combat_defeat") || user.HasQuest("1-crafting_arrive") ) {
        return true;
    }

    var smith = room.GetMob(smithMobId, true);
    if ( smith == null ) {
        return true;
    }

    // Give crafting materials
    user.GiveItem(ironIngotId);
    user.GiveItem(leatherStripId);

    smith.Command('emote sets down the hammer and turns to face you. The forge has been burning continuously for decades -- the heat in this room is a permanent condition.', 1.0);
    smith.Command('say Crafting is belief made material. You take raw substance and impose a shape on it through skill and intent.', 2.0);
    smith.Command('say The forge does not care about your feelings. It cares about your technique.', 3.5);
    smith.Command('say Heat the metal until it tells you it is ready -- color, texture, the sound it makes. Then shape it before doubt sets in.', 5.0);
    smith.Command('say I have put an iron ingot and a leather strip in your pack.', 6.5);
    smith.Command('say Type <ansi fg="command">craft</ansi> to see your options. Then type <ansi fg="command">craft iron dagger</ansi>. One attempt. Go.', 8.0);

    user.GiveQuest("1-crafting_arrive");
    craftStarted   = false;
    craftIdleTicks = 0;

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Detect when the player actually starts crafting (not just listing recipes)
    if ( cmd == "craft" && rest != "" && rest != "list" ) {
        if ( user.HasQuest("1-crafting_arrive") && !user.HasQuest("1-crafting_craft") ) {
            craftStarted   = true;
            craftIdleTicks = 0;
        }
    }

    return false;
}

function onIdle(room) {

    if ( !craftStarted ) {
        return;
    }

    craftIdleTicks++;

    // Iron dagger takes 3 rounds; wait 5 ticks to be sure it has resolved
    if ( craftIdleTicks < 5 ) {
        return;
    }

    craftStarted   = false;
    craftIdleTicks = 0;

    var players = room.GetPlayers();
    if ( players == null ) {
        return;
    }

    for ( var i = 0; i < players.length; i++ ) {
        var player = players[i];
        if ( !player.HasQuest("1-crafting_arrive") || player.HasQuest("1-crafting_craft") ) {
            continue;
        }

        var smith = room.GetMob(smithMobId, true);

        if ( player.HasItemId(ironDaggerId) ) {
            // Success
            if ( smith != null ) {
                smith.Command('emote glances at the finished dagger and gives a single slow nod.', 0.5);
                smith.Command('say Acceptable. You have the idea.', 1.5);
                smith.Command('say The Alchemist is west of the well court. She works with a different kind of precision.', 2.5);
                smith.Command('say If you get turned around, type <ansi fg="command">map</ansi> -- it will show you where you are.', 4.0);
            }
        } else {
            // Failure - materials consumed
            if ( smith != null ) {
                smith.Command('emote watches the result without expression.', 0.5);
                smith.Command('emote mutters something under their breath and sets the failure aside.', 1.5);
                smith.Command('say The principle is sound. The body needs repetition to trust it.', 2.5);
                smith.Command('say The Alchemist is west of the well court. Different materials, same lesson.', 3.5);
                smith.Command('say If you get turned around, type <ansi fg="command">map</ansi> -- it will show you where you are.', 5.0);
            }
        }

        player.GiveQuest("1-crafting_craft");
    }
}

function onExit(user, room) {
    craftStarted   = false;
    craftIdleTicks = 0;
}

function onLoad(room) {
    craftStarted   = false;
    craftIdleTicks = 0;
}
