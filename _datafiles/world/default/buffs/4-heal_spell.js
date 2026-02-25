
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'A magical healing aura washes over you.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is surrounded by a healing glow.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.floor(maxHP * 0.05);
    if (healAmt < 1) { healAmt = 1; }
    actor.AddHealth(healAmt);

    SendUserMessage(actor.UserId(),     'The healing aura mends your wounds.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is healing from the effects of a heal spell.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'The healing aura fades away.');
    SendRoomMessage(actor.GetRoomId(),  'The healing aura surrounding '+actor.GetCharacterName(true)+' fades away.', actor.UserId());
}
