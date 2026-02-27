function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Your muscles twitch uncontrollably as nerve signals misfire.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Your nerves settle and control returns to your limbs.');
}
