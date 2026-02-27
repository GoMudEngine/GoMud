function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Your mind opens and sharpens — every action feels more instructive.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The heightened attunement fades from your mind.');
}
