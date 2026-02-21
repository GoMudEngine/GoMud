
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The compress warms against your skin as the herbs do their work.');
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    actor.AddHealth(UtilDiceRoll(2, 3));
    SendUserMessage(actor.UserId(), 'You feel a gentle mending wash over you.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' is healing from the effects of a poultice.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The poultice\'s warmth fades.');
}
