function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Chrysalis energy surges through your cells, accelerating mutation.');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The mutagenic catalyst fades from your system.');
}
