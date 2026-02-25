function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Your senses explode with input — everything is too bright, too loud!');
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The sensory overload subsides and your senses normalize.');
}
