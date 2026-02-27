function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Chrysalis energy suffuses your body with a warm, mending pulse.');
}

function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.max(1, Math.floor(maxHP * 0.05));
    actor.AddHealth(healAmt);
    SendUserMessage(actor.UserId(), 'The vital surge mends your wounds.');
}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The vital surge fades.');
}
