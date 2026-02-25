function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Chrysalis energy floods your nerves — everything feels faster.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The Chrysalis haste fades and your movements return to normal.');
}
