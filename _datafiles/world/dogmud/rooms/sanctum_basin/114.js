// Training Yard - Room 114
// Combat Trainer gives combat instruction and checks if training dummy is defeated.

const trainerMobId = 51;
const dummyMobId = 65;

var dummySpawnedForUser = false;

function onEnter(user, room) {

    // Only trigger if player has combat quest and hasn't completed it
    if ( !user.HasQuest("1-combat") || user.HasQuest("1-crafting") ) {
        return false;
    }

    var trainer = room.GetMob(trainerMobId, true);
    if ( trainer == null ) {
        return false;
    }

    // Ensure training dummy is present
    var dummy = room.GetMob(dummyMobId, true);

    trainer.Command('say You are here for combat training.', 1.0);
    trainer.Command('say The fundamentals are simple: close distance, maintain balance, apply force. Everything else is variation.', 2.5);
    trainer.Command('say The training dummy will not fight back, but it will give you a sense of how your body moves under pressure.', 4.0);
    trainer.Command('say Strike it. Use the <ansi fg="command">attack dummy</ansi> command. Observe what happens.', 5.5);

    dummySpawnedForUser = true;

    return false;
}

function onIdle(room) {

    // Check if dummy has been defeated (no longer present) and player is in progress
    if ( !dummySpawnedForUser ) {
        return;
    }

    var dummy = room.GetMob(dummyMobId, false);
    if ( dummy != null ) {
        return; // Dummy still alive
    }

    // Dummy is gone - find players in room who have the combat quest pending
    var players = room.GetPlayers();
    if ( players == null ) {
        dummySpawnedForUser = false;
        return;
    }

    for ( var i = 0; i < players.length; i++ ) {
        var player = players[i];
        if ( player.HasQuest("1-combat") && !player.HasQuest("1-crafting") ) {

            var trainer = room.GetMob(trainerMobId, true);
            if ( trainer != null ) {
                trainer.Command('say Well done. You have a sense of it now.', 1.0);
                trainer.Command('say The Forge is south of here. Speak with Korvath -- physical mastery and craft mastery are not separate things.', 2.5);
            }

            player.GiveQuest("1-crafting");
        }
    }

    dummySpawnedForUser = false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var trainer = room.GetMob(trainerMobId, true);
        if ( trainer != null ) {
            if ( user.HasQuest("1-combat") && !user.HasQuest("1-crafting") ) {
                trainer.Command('say Attack the training dummy. Type <ansi fg="command">attack dummy</ansi>.');
            } else {
                trainer.Command('say Keep practicing. The body learns through repetition, not instruction.');
            }
        }
        return false;
    }
    return false;
}

function onExit(user, room) {
    dummySpawnedForUser = false;
}

function onLoad(room) {
    dummySpawnedForUser = false;
}
