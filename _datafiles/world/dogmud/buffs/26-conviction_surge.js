function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Conviction surges through your limbs, empowering your strikes.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The surge of conviction fades from your limbs.');
}
