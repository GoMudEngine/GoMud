// Chrysalis Phantom - Mob 272
// Boss mob in the Chrysalis Undercity. Fights with hit-and-run tactics.
// BANDAID: after fleeing, re-sneaks + re-hides, then tracks back and
// backstabs for repeated surprise strike rounds.

var hasEngaged = false;
var fleeRecovery = 0;
var homeRoom = 508; // Chitin Throne

function onHurt(mob, room, eventDetails) {
    hasEngaged = true;
    return false;
}

function onIdle(mob, room) {
    // After fleeing, count down then track back
    if (fleeRecovery > 0) {
        fleeRecovery--;
        if (fleeRecovery === 2) {
            // Force hidden buff (buff 9) so we get surprise strike
            mob.Command('sneak');
            mob.AddBuff(9, 'self');
        }
        if (fleeRecovery <= 0) {
            // Go back to home room
            mob.Command('go south');
        }
        return true;
    }

    if (!hasEngaged) { return false; }

    // If not fighting and players are in the room, surprise attack
    if (!mob.IsAggroed()) {
        var players = room.GetPlayers();
        if (players && players.length > 0) {
            mob.Command('attack');
        }
    }

    return false;
}

function onCommand_flee(rest, mob, room, eventDetails) {
    // Set up recovery: 3 ticks (sneak on tick 2, move on tick 0)
    fleeRecovery = 3;
    return false; // let the flee happen
}
