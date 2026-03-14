// Bandit Leader - Mob 254
// Spawns non-hostile. Starts aggro countdown on player enter.
// If player initiates dialogue within 5 rounds, countdown cancels.
// If countdown expires, leader attacks.
// The leader's entrance line is handled by the dialogue tree root text.

var aggroCountdown = 0;
var targetUserId = 0;
var negotiating = false;

function onPlayerEnter(mob, room, eventDetails) {
    // Don't restart countdown if already negotiating
    if ( negotiating ) { return true; }

    aggroCountdown = 5;
    targetUserId = eventDetails.sourceId;
    return true;
}

function onIdle(mob, room) {
    if ( aggroCountdown <= 0 || negotiating ) { return false; }

    aggroCountdown--;

    if ( aggroCountdown == 2 ) {
        mob.Command('say Last chance. Pay the toll or walk away. Or do not. Your choice.');
    }

    if ( aggroCountdown <= 0 && !negotiating ) {
        mob.Command('attack');
    }

    return false;
}

function onAsk(mob, room, eventDetails) {
    // Any dialogue attempt cancels the countdown
    negotiating = true;
    aggroCountdown = 0;
    return false; // Let the dialogue system handle the response
}
