
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The tonic sharpens your thoughts to a razor\'s edge.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + '\'s eyes brighten with renewed focus.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The mental clarity fades, leaving your thoughts slightly muddled.');
}
