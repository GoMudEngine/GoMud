// 
// buff zero (0) is a special buff that when naturally expires, 
// will remove the player from the game without zombie status.
//

// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),    'You sit down and begin your meditation.' );
    SendUserMessage(actor.UserId(),    'Your meditation must complete without interruption to quit gracefully.');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' sits down a begins to meditate.', actor.UserId());
}

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You continue your meditation. <ansi bg="blue"> *' + triggersLeft + ' rounds left* </ansi>' );
    SendRoomMessage(actor.GetRoomId(),   actor.GetCharacterName(true)+' continues meditating.', actor.UserId() );
}
