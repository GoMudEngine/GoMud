// Stunned buff script

function onStart(actor) {
    SendUserMessage(actor.UserId(), '<ansi fg="yellow">You are stunned! You cannot attack.</ansi>');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' staggers, dazed!', actor.UserId());
}

function onEnd(actor) {
    SendUserMessage(actor.UserId(), '<ansi fg="green">You shake off the stun.</ansi>');
}

function onTrigger(actor) {}
