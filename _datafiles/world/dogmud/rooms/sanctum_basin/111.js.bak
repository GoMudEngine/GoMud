// The Alchemist's Workshop - Room 111
// Alchemist Yenna gives mats, player crafts healing salve, Yenna reacts to outcome.
// Sub-steps: alchemy_arrive -> alchemy_craft

const alchemistMobId = 53;
const healersRootId  = 40004;
const glassVialId    = 40006;
const salveId        = 30036;

var craftStarted    = false;
var craftIdleTicks  = 0;

function onEnter(user, room) {

    // Only trigger if player has done crafting_craft but not yet alchemy_arrive
    if ( !user.HasQuest("1-crafting_craft") || user.HasQuest("1-alchemy_arrive") ) {
        return true;
    }

    var alchemist = room.GetMob(alchemistMobId, true);
    if ( alchemist == null ) {
        return true;
    }

    // Give crafting materials
    user.GiveItem(healersRootId);
    user.GiveItem(healersRootId);
    user.GiveItem(glassVialId);

    alchemist.Command('say Alchemy is pattern recognition. The world has rules and compounds have relationships.', 1.0);
    alchemist.Command('say A healing salve is ground healer\'s root mixed with water and sealed in a vial. Simple, but effective.', 2.5);
    alchemist.Command('say I have put two healer\'s roots and a glass vial in your pack.', 4.5);
    alchemist.Command('say Type <ansi fg="command">craft</ansi> to see your options. Then type <ansi fg="command">craft healing salve</ansi>. One attempt.', 6.0);

    user.GiveQuest("1-alchemy_arrive");
    craftStarted   = false;
    craftIdleTicks = 0;

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Detect when the player actually starts crafting (not just listing recipes)
    if ( cmd == "craft" && rest != "" && rest != "list" ) {
        if ( user.HasQuest("1-alchemy_arrive") && !user.HasQuest("1-alchemy_craft") ) {
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

    // Healing salve takes 3 rounds; wait 5 ticks to be sure it has resolved
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
        if ( !player.HasQuest("1-alchemy_arrive") || player.HasQuest("1-alchemy_craft") ) {
            continue;
        }

        var hasSalve = player.HasItemId(salveId);
        var hasRoot  = player.HasItemId(healersRootId);
        var hasVial  = player.HasItemId(glassVialId);

        // Only grant quest if crafting actually happened:
        // Success: player has the salve
        // Failure: materials were consumed (missing root or vial)
        // If they still have all materials, the craft was rejected (wrong recipe/station)
        if ( hasRoot && hasVial && !hasSalve ) {
            // Craft was rejected — don't grant quest, let them try again
            var npc = room.GetMob(alchemistMobId, true);
            if ( npc != null ) {
                npc.Command('say That didn\'t take. Check the recipe and try again. Type <ansi fg="command">craft healing salve</ansi>.', 0.5);
            }
            continue;
        }

        var alchemist = room.GetMob(alchemistMobId, true);

        if ( hasSalve ) {
            // Success
            if ( alchemist != null ) {
                alchemist.Command('emote examines the salve with a brief, approving look.', 0.5);
                alchemist.Command('say Good. The color is right. You understood what you were doing.', 1.5);
                alchemist.Command('say To use it: type <ansi fg="command">drink salve</ansi>. It applies a regen effect -- you will see it listed when you type <ansi fg="command">condition</ansi>.', 2.5);
                alchemist.Command('say Find Fen in the West Meadow -- go south from here.', 4.0);
            }
        } else {
            // Failure - materials consumed
            if ( alchemist != null ) {
                alchemist.Command('emote frowns at the ruined mixture without comment.', 0.5);
                alchemist.Command('say The world is full of things that want to kill you. Knowing what heals you is not optional.', 1.5);
                alchemist.Command('say When you do have a consumable, type <ansi fg="command">drink [item]</ansi> to use it. Type <ansi fg="command">condition</ansi> to see what effects are active on you.', 3.0);
                alchemist.Command('say Find Fen in the West Meadow -- go south from here.', 4.5);
            }
        }

        player.GiveQuest("1-alchemy_craft");
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
