function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Solidified conviction wraps around your body like armor.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' is encased in a shimmering layer of conviction.', actor.UserId());
}

function onTrigger(actor, triggersLeft) {}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The conviction armor fades from your body.');
}
