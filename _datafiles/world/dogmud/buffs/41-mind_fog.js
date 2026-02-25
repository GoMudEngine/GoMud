function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'A thick fog descends over your thoughts...');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The fog lifts from your mind and clarity returns.');
}
