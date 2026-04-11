function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.max(1, Math.floor(maxHP * 0.08));
    actor.AddHealth(healAmt);
    SendUserMessage(actor.UserId(), 'The Chrysalis regeneration mends your wounds rapidly.');
}
