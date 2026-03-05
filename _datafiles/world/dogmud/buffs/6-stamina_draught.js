
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The bitter draught burns going down, then spreads warmth through your legs.');
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    var maxSP = actor.GetStaminaMax();
    var restoreAmt = Math.floor(maxSP * 0.08);
    if (restoreAmt < 1) { restoreAmt = 1; }
    actor.AddStamina(restoreAmt);
    SendUserMessage(actor.UserId(), 'Your legs feel stronger, your breath steadier.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' seems to recover some energy.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The draught\'s effect fades.');
}
