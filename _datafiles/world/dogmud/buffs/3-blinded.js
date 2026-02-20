// Blinded buff script

function onStart(actor) {
    SendUserMessage(actor.UserId(), '<ansi fg="yellow">Your vision goes dark — you have been blinded!</ansi>');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true)+' claws at their eyes, blinded!', actor.UserId());
}

function onEnd(actor) {
    SendUserMessage(actor.UserId(), '<ansi fg="green">Your vision slowly returns to normal.</ansi>');
}

function onTrigger(actor) {}
