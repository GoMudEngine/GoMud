
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'A warm tingling spreads through your body as you become more resistant to heat.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + '\'s skin takes on a faint reddish hue.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The protective warmth fades from your skin.');
}
