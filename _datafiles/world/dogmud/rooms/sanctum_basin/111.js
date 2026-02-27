// The Alchemist's Workshop - Room 111
// Alchemist Yenna gives mats, player crafts healing poultice, Yenna reacts to outcome.
// Sub-steps: alchemy_arrive -> alchemy_craft

const alchemistMobId = 53;
const healersRootId  = 40004;
const clothStripId   = 40007;
const poulticeId     = 30010;

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
    user.GiveItem(clothStripId);

    alchemist.Command('say Alchemy is pattern recognition. The world has rules and compounds have relationships.', 1.0);
    alchemist.Command('say A healing poultice is crushed healer\'s root, pressed into cloth. The root draws contamination. The cloth carries it.', 2.5);
    alchemist.Command('say Everything in that formulation has a reason. Swap components without understanding why and you get something useless at best, poisonous at worst.', 4.5);
    alchemist.Command('say I have put two healer\'s roots and a cloth strip in your pack.', 6.0);
    alchemist.Command('say Type <ansi fg="command">craft</ansi> to see your options. Then type <ansi fg="command">craft healing poultice</ansi>. One attempt.', 7.5);

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

    // Healing poultice takes 2 rounds; wait 4 ticks to be sure it has resolved
    if ( craftIdleTicks < 4 ) {
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

        var alchemist = room.GetMob(alchemistMobId, true);

        if ( player.HasItemId(poulticeId) ) {
            // Success
            if ( alchemist != null ) {
                alchemist.Command('emote examines the poultice with a brief, approving look.', 0.5);
                alchemist.Command('say Good. The color is right. You understood what you were doing.', 1.5);
                alchemist.Command('say To use it: type <ansi fg="command">drink poultice</ansi>. It applies a regen effect -- you will see it listed when you type <ansi fg="command">condition</ansi>.', 2.5);
                alchemist.Command('say The condition command shows everything currently affecting you. Buffs, poisons, regen ticks -- all of it. Check it when something feels wrong.', 4.0);
                alchemist.Command('say Find Fen in the West Meadow, west of Town Square. They will teach you what grows out there.', 5.5);
            }
        } else {
            // Failure - materials consumed
            if ( alchemist != null ) {
                alchemist.Command('emote frowns at the ruined mixture without comment.', 0.5);
                alchemist.Command('emote makes a quiet note in the ledger, expression unreadable.', 1.5);
                alchemist.Command('say The world is full of things that want to kill you. Knowing what heals you is not optional.', 2.5);
                alchemist.Command('say When you do have a consumable, type <ansi fg="command">drink [item]</ansi> to use it. Type <ansi fg="command">condition</ansi> to see what effects are active on you.', 4.0);
                alchemist.Command('say Find Fen in the West Meadow, west of Town Square. They will show you where the raw materials come from.', 5.5);
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
