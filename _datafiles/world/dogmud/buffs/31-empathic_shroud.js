function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'You wrap yourself in a psionic shroud, fading from perception.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' seems to shimmer and fade from view.', actor.UserId());
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The empathic shroud dissolves and you become visible again.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' shimmers back into view.', actor.UserId());
}
