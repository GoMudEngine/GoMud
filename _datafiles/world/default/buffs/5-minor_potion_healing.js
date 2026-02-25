
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The potion warms you as you drink it down.');
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.floor(maxHP * 0.04);
    if (healAmt < 1) { healAmt = 1; }
    actor.AddHealth(healAmt);

    SendUserMessage(actor.UserId(),     'The potion mends your wounds.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is healing from the effects of a potion.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The potions effect runs out.');
}
