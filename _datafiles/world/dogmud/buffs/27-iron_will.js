function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Your mind hardens like iron, resolute and unyielding.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The iron resolve in your mind softens back to normal.');
}
