
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'Raw power surges through your muscles as the elixir takes hold. Your hands tremble with barely contained fury.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + '\'s muscles bulge and veins stand out as a wild look enters their eyes.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The berserker rage subsides, leaving you feeling drained but steady.');
}
