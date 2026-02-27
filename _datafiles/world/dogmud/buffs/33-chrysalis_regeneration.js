function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Deep Chrysalis energy wraps around you, accelerating your body\'s healing.');
}

function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.max(1, Math.floor(maxHP * 0.08));
    actor.AddHealth(healAmt);
    SendUserMessage(actor.UserId(), 'The Chrysalis regeneration mends your wounds rapidly.');
}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The Chrysalis regeneration fades.');
}
