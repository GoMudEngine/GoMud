
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The sharp draught clears your thoughts as you swallow it.');
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    actor.AddConviction(UtilDiceRoll(2, 3));
    SendUserMessage(actor.UserId(), 'Your thoughts clear, your resolve hardens.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' seems to steel their mind.', actor.UserId());
}

// Invoked when the buff has run its course.
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The draught\'s clarity fades.');
}
