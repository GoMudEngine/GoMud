
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The poultice floods your body with deep, restorative warmth.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' sighs with relief as herbs do their work.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.floor(maxHP * 0.12);
    if (healAmt < 1) { healAmt = 1; }
    actor.AddHealth(healAmt);

    SendUserMessage(actor.UserId(), 'You feel a strong wave of healing wash over you.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' is healing from the effects of a potent poultice.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The deep warmth of the poultice fades away.');
}
