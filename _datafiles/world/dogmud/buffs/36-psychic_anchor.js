function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'A psionic anchor locks onto your mind, rooting you in place!');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The psychic anchor releases its grip on your mind.');
}
