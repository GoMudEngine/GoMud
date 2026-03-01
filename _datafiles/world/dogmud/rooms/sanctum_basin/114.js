// Training Yard - Room 114
// Combat Trainer gives combat instruction and checks if training dummy is defeated.
// Sub-steps: combat_arrive -> combat_defeat

const trainerMobId = 51;
const dummyMobId = 65;

var trainerHasCommented = false;

function onEnter(user, room) {

    // Only trigger if player has completed shopping_equip but hasn't arrived yet
    if ( !user.HasQuest("1-shopping_equip") || user.HasQuest("1-combat_arrive") ) {
        return true;
    }

    var trainer = room.GetMob(trainerMobId, true);
    if ( trainer == null ) {
        return true;
    }

    // Ensure training dummy is present
    var dummy = room.GetMob(dummyMobId, true);

    trainer.Command('say You are here for combat training.', 1.0);
    trainer.Command('say The fundamentals are simple: close distance, maintain balance, apply force. Everything else is variation.', 2.5);
    trainer.Command('say The training dummy is an easy opponent. It will not overwhelm you.', 4.0);
    trainer.Command('say Defeat it. Use <ansi fg="command">attack dummy</ansi> to start the fight. Do not stop until it falls.', 5.5);
    trainer.Command('emote watches with arms folded, the faintest smile on their face.', 6.5);

    user.GiveQuest("1-combat_arrive");

    return true;
}

function onIdle(room) {

    // Check if any player needs to defeat the dummy
    var players = room.GetPlayers();
    if ( players == null ) {
        return;
    }

    var hasEligiblePlayer = false;
    for ( var i = 0; i < players.length; i++ ) {
        if ( players[i].HasQuest("1-combat_arrive") && !players[i].HasQuest("1-combat_defeat") ) {
            hasEligiblePlayer = true;
            break;
        }
    }

    if ( !hasEligiblePlayer ) {
        return;
    }

    var dummy = room.GetMob(dummyMobId, false);

    if ( dummy != null ) {
        // Dummy is still alive — comment once when player has hit it
        if ( !trainerHasCommented && dummy.GetHealth() < dummy.GetHealthMax() ) {
            trainerHasCommented = true;
            var trainer = room.GetMob(trainerMobId, false);
            if ( trainer != null ) {
                trainer.Command('emote lets out a short laugh.', 0.5);
                trainer.Command('say I may have understated what the dummy is capable of. Keep going.', 1.5);
                trainer.Command('say While you are at it -- combat is not just striking. Try <ansi fg="command">kick</ansi>, <ansi fg="command">grapple</ansi>, or <ansi fg="command">trip</ansi> to mix it up.', 3.5);
                trainer.Command('say If you had a shield, <ansi fg="command">bash</ansi> is also available -- shield work comes later. For now, get it on the ground.', 5.0);
                trainer.Command('say One more thing. Not every fight is won with a blade. Try <ansi fg="command">taunt</ansi> -- a sharp tongue can break an opponent as surely as a sharp edge.', 7.0);
            }
        }
        return;
    }

    // Dummy is gone - grant combat_defeat to eligible players
    for ( var i = 0; i < players.length; i++ ) {
        var player = players[i];
        if ( player.HasQuest("1-combat_arrive") && !player.HasQuest("1-combat_defeat") ) {

            var trainer = room.GetMob(trainerMobId, true);
            if ( trainer != null ) {
                trainer.Command('say Well done. You have a sense of it now.', 1.0);
                trainer.Command('say The Forge is south of here. Speak with Korvath -- physical mastery and craft mastery are not separate things.', 2.5);
            }

            player.GiveQuest("1-combat_defeat");
        }
    }

    trainerHasCommented = false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var trainer = room.GetMob(trainerMobId, true);
        if ( trainer != null ) {
            if ( user.HasQuest("1-combat_arrive") && !user.HasQuest("1-combat_defeat") ) {
                trainer.Command('say Defeat the training dummy. Type <ansi fg="command">attack dummy</ansi> to start the fight.');
            } else {
                trainer.Command('say Keep practicing. The body learns through repetition, not instruction.');
            }
        }
        return false;
    }
    return false;
}

function onExit(user, room) {
}

function onLoad(room) {
    trainerHasCommented = false;
}
