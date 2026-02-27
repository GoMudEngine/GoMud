
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.floor(maxHP * 0.05);
    if (healAmt < 1) { healAmt = 1; }
    actor.AddHealth(healAmt);

    actor.RemoveBuff(39);
    actor.RemoveBuff(40);

    SendUserMessage(actor.UserId(), 'The bitter draught burns your throat but you feel poisons loosening their grip.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' grimaces as they swallow an antidote.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The antidote has run its course.');
}
