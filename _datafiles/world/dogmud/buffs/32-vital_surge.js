function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.max(1, Math.floor(maxHP * 0.05));
    actor.AddHealth(healAmt);
    SendUserMessage(actor.UserId(), 'The vital surge mends your wounds.');
}
