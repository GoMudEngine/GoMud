
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'A warm glow surrounds you.');
    SendRoomMessage(actor.GetRoomId(),  'A warm glow surrounds '+actor.GetCharacterName(true)+ '.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'Your glowing fades away.' );
    SendRoomMessage(actor.GetRoomId(),  'The glow surrounding '+actor.GetCharacterName(true)+ ' fades away.', actor.UserId());
}
