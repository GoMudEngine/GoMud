
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You feel sneaky.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' disappears into the shadows.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You no longer feel sneaky.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' emerges from the shadows.', actor.UserId());
}
